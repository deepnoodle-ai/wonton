// Package sse provides a parser and client for Server-Sent Events (SSE) streams.
//
// Server-Sent Events (SSE) is a standard for streaming data from servers to clients
// over HTTP. It is commonly used for streaming responses from LLMs like OpenAI and
// Anthropic, real-time notifications, live updates, and other event-driven applications.
//
// # Reading SSE Streams
//
// The Reader provides low-level access to parse SSE streams from any io.Reader:
//
//	resp, _ := http.Get(url)
//	defer resp.Body.Close()
//	reader := sse.NewReader(resp.Body)
//
//	for {
//	    event, err := reader.Read()
//	    if err == io.EOF {
//	        break
//	    }
//	    if err != nil {
//	        return err
//	    }
//	    fmt.Println(event.Data)
//	}
//
// For simpler iteration, use the callback-based Stream function:
//
//	err := sse.Stream(resp.Body, func(event sse.Event) error {
//	    fmt.Println(event.Data)
//	    return nil // Return error to stop streaming
//	})
//
// # HTTP Client
//
// The Client handles HTTP connections and automatically sets required headers:
//
//	client := sse.NewClient("https://api.example.com/stream")
//	client.Headers.Set("Authorization", "Bearer token")
//
//	ctx := context.Background()
//	events, errs := client.Connect(ctx)
//
//	for event := range events {
//	    fmt.Println(event.Data)
//	}
//
//	if err := <-errs; err != nil {
//	    log.Fatal(err)
//	}
//
// # Event Format
//
// Events are parsed according to the SSE specification (https://html.spec.whatwg.org/multipage/server-sent-events.html).
// Each event may contain:
//   - event: The event type (optional, defaults to "message")
//   - data: The payload (required, may span multiple lines)
//   - id: An event ID for resuming streams (optional)
//   - retry: Reconnection time in milliseconds (optional)
//
// Multi-line data fields are joined with newlines. Events are delimited by blank lines.
// Lines starting with ":" are comments and ignored.
//
// # Parsing Utilities
//
// For testing or working with buffered SSE data, use ParseString or ParseBytes:
//
//	events, err := sse.ParseString(`data: hello\n\ndata: world\n\n`)
//	for _, event := range events {
//	    fmt.Println(event.Data)
//	}
package sse

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// Event represents a single Server-Sent Event.
//
// According to the SSE specification, an event consists of one or more fields.
// The Data field contains the actual payload and is typically required. The Event
// field specifies the event type, the ID field can be used for stream resumption,
// and the Retry field suggests a reconnection interval.
type Event struct {
	// ID is the event ID (optional). Used by clients to resume streams by sending
	// this ID in the Last-Event-ID header.
	ID string

	// Event is the event type (optional, defaults to "message"). Clients can filter
	// events by type.
	Event string

	// Data is the event payload. For multi-line data fields, lines are joined with
	// newlines. This is typically JSON-encoded data but can be any string.
	Data string

	// Retry is the reconnection time in milliseconds (optional). Suggests how long
	// the client should wait before reconnecting if the connection is lost.
	Retry int
}

// JSON unmarshals the event data as JSON into v.
//
// This is a convenience method for events with JSON payloads, which are common
// in SSE streams from LLM APIs and other modern services.
//
// Example:
//
//	var data struct {
//	    Message string `json:"message"`
//	}
//	if err := event.JSON(&data); err != nil {
//	    return err
//	}
//	fmt.Println(data.Message)
func (e *Event) JSON(v any) error {
	return json.Unmarshal([]byte(e.Data), v)
}

// IsEmpty returns true if the event has no meaningful content.
//
// An event is considered empty if it has no data, no event type, and no ID.
// Empty events can occur when parsing streams with only comments or blank lines.
func (e *Event) IsEmpty() bool {
	return e.Data == "" && e.Event == "" && e.ID == ""
}

// Reader reads Server-Sent Events from an io.Reader.
//
// Reader provides low-level parsing of SSE streams. It reads events line-by-line
// and assembles them according to the SSE specification. For HTTP-based SSE
// connections with automatic reconnection, use Client instead.
type Reader struct {
	scanner *bufio.Scanner
	event   Event
}

// NewReader creates a new SSE reader that parses events from r.
//
// The reader buffers input and parses events according to the SSE specification.
// Call Read repeatedly to get each event from the stream.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

// Buffer sets the maximum buffer size for reading lines.
//
// The default maximum line size is 64KB. Call this method before the first Read
// if you need to handle events with longer lines, such as large JSON payloads
// or base64-encoded data.
//
// Example:
//
//	reader := sse.NewReader(resp.Body)
//	reader.Buffer(1024 * 1024) // 1MB buffer
//	event, err := reader.Read()
func (r *Reader) Buffer(maxLineSize int) {
	r.scanner.Buffer(make([]byte, 0, maxLineSize), maxLineSize)
}

// Read reads and returns the next event from the stream.
//
// Read blocks until a complete event is available or an error occurs. Events are
// delimited by blank lines in the stream. Returns io.EOF when the stream ends
// normally.
//
// Example:
//
//	reader := sse.NewReader(resp.Body)
//	for {
//	    event, err := reader.Read()
//	    if err == io.EOF {
//	        break
//	    }
//	    if err != nil {
//	        return err
//	    }
//	    fmt.Printf("Event: %s, Data: %s\n", event.Event, event.Data)
//	}
func (r *Reader) Read() (Event, error) {
	var dataLines []string

	for r.scanner.Scan() {
		line := r.scanner.Text()

		// Empty line marks end of event
		if line == "" {
			if len(dataLines) > 0 || r.event.Event != "" || r.event.ID != "" {
				r.event.Data = strings.Join(dataLines, "\n")
				event := r.event
				r.event = Event{} // Reset for next event
				return event, nil
			}
			continue
		}

		// Parse the line
		if strings.HasPrefix(line, ":") {
			// Comment, ignore
			continue
		}

		field, value, _ := strings.Cut(line, ":")
		// Remove leading space from value (per SSE spec)
		value = strings.TrimPrefix(value, " ")

		switch field {
		case "event":
			r.event.Event = value
		case "data":
			dataLines = append(dataLines, value)
		case "id":
			r.event.ID = value
		case "retry":
			// Parse retry as integer milliseconds
			if retry, err := strconv.Atoi(value); err == nil && retry >= 0 {
				r.event.Retry = retry
			}
		}
	}

	if err := r.scanner.Err(); err != nil {
		return Event{}, err
	}

	// Check if we have a partial event at EOF
	if len(dataLines) > 0 || r.event.Event != "" || r.event.ID != "" {
		r.event.Data = strings.Join(dataLines, "\n")
		event := r.event
		r.event = Event{}
		return event, nil
	}

	return Event{}, io.EOF
}

// Stream reads all events from r and calls fn for each event.
//
// Stream is a convenience function that combines event reading with iteration.
// It continues until the stream ends (io.EOF) or fn returns an error. If fn
// returns an error, streaming stops and that error is returned.
//
// Example:
//
//	resp, _ := http.Get(url)
//	defer resp.Body.Close()
//
//	err := sse.Stream(resp.Body, func(event sse.Event) error {
//	    fmt.Println(event.Data)
//	    if event.Data == "DONE" {
//	        return io.EOF // Stop processing
//	    }
//	    return nil
//	})
func Stream(r io.Reader, fn func(Event) error) error {
	reader := NewReader(r)
	for {
		event, err := reader.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := fn(event); err != nil {
			return err
		}
	}
}

// Client is an HTTP client for Server-Sent Events streams.
//
// Client handles the HTTP connection to an SSE endpoint, automatically setting
// required headers (Accept, Cache-Control, Connection) and managing the Last-Event-ID
// header for stream resumption. For lower-level parsing of SSE data from any source,
// use Reader instead.
type Client struct {
	// URL is the SSE endpoint to connect to.
	URL string

	// Headers are additional HTTP headers to send with requests.
	// Common use cases include authorization headers.
	Headers http.Header

	// HTTPClient is the underlying HTTP client to use for requests.
	// If nil, http.DefaultClient is used.
	HTTPClient *http.Client

	// LastEventID is the most recently received event ID.
	// It is sent as the Last-Event-ID header on subsequent connections,
	// allowing the server to resume the stream from where it left off.
	LastEventID string
}

// NewClient creates a new SSE client for the given URL.
//
// The client is ready to use immediately. Set Headers before calling Connect
// to add custom headers like authorization tokens.
//
// Example:
//
//	client := sse.NewClient("https://api.example.com/stream")
//	client.Headers.Set("Authorization", "Bearer token123")
func NewClient(url string) *Client {
	return &Client{
		URL:     url,
		Headers: make(http.Header),
	}
}

// Connect establishes an SSE connection and returns channels for events and errors.
//
// Connect makes an HTTP GET request to the configured URL with appropriate SSE headers.
// Events are delivered on the returned event channel, and any errors are sent to the
// error channel. Both channels are closed when the connection ends or ctx is cancelled.
//
// The connection validates that the response has status 200 OK and Content-Type
// text/event-stream. If LastEventID is set, it is sent in the Last-Event-ID header.
//
// Example:
//
//	client := sse.NewClient(url)
//	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
//	defer cancel()
//
//	events, errs := client.Connect(ctx)
//	for event := range events {
//	    fmt.Printf("Received: %s\n", event.Data)
//	}
//
//	if err := <-errs; err != nil && err != context.DeadlineExceeded {
//	    log.Printf("Error: %v", err)
//	}
func (c *Client) Connect(ctx context.Context) (<-chan Event, <-chan error) {
	events := make(chan Event)
	errs := make(chan error, 1)

	go c.run(ctx, events, errs)

	return events, errs
}

func (c *Client) run(ctx context.Context, events chan<- Event, errs chan<- error) {
	defer close(events)
	defer close(errs)

	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.URL, nil)
	if err != nil {
		errs <- err
		return
	}

	// Set SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Copy custom headers
	for k, v := range c.Headers {
		req.Header[k] = v
	}

	// Set Last-Event-ID if available
	if c.LastEventID != "" {
		req.Header.Set("Last-Event-ID", c.LastEventID)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		errs <- err
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errs <- &HTTPError{StatusCode: resp.StatusCode, Status: resp.Status}
		return
	}

	// Validate Content-Type (should be text/event-stream, possibly with charset)
	ct := resp.Header.Get("Content-Type")
	if ct != "" && !strings.HasPrefix(ct, "text/event-stream") {
		errs <- &HTTPError{StatusCode: resp.StatusCode, Status: "unexpected content-type: " + ct}
		return
	}

	reader := NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			errs <- ctx.Err()
			return
		default:
		}

		event, err := reader.Read()
		if err == io.EOF {
			return
		}
		if err != nil {
			errs <- err
			return
		}

		// Update Last-Event-ID
		if event.ID != "" {
			c.LastEventID = event.ID
		}

		select {
		case events <- event:
		case <-ctx.Done():
			errs <- ctx.Err()
			return
		}
	}
}

// HTTPError represents an HTTP error response from an SSE endpoint.
//
// HTTPError is returned by Client.Connect when the server returns a non-200
// status code or when the Content-Type is not text/event-stream.
type HTTPError struct {
	// StatusCode is the HTTP status code (e.g., 401, 404, 500).
	StatusCode int

	// Status is a human-readable error message, typically the HTTP status text
	// or a description of the validation error.
	Status string
}

func (e *HTTPError) Error() string {
	return "sse: " + e.Status
}

// ParseBytes parses SSE events from a byte slice.
//
// This function is useful for testing or when working with buffered SSE data.
// It returns all events found in the data or an error if parsing fails.
//
// Example:
//
//	data := []byte("data: hello\n\ndata: world\n\n")
//	events, err := sse.ParseBytes(data)
//	for _, event := range events {
//	    fmt.Println(event.Data)
//	}
func ParseBytes(data []byte) ([]Event, error) {
	var events []Event
	err := Stream(bytes.NewReader(data), func(event Event) error {
		events = append(events, event)
		return nil
	})
	return events, err
}

// ParseString parses SSE events from a string.
//
// This is a convenience wrapper around ParseBytes for string input.
// It returns all events found in the string or an error if parsing fails.
//
// Example:
//
//	events, err := sse.ParseString("event: ping\ndata: {}\n\n")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(events[0].Event) // Output: ping
func ParseString(data string) ([]Event, error) {
	return ParseBytes([]byte(data))
}
