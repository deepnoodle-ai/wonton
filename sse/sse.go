// Package sse provides a parser for Server-Sent Events (SSE) streams.
// SSE is commonly used for streaming responses from LLMs like OpenAI and Anthropic.
//
// Basic usage:
//
//	resp, _ := http.Get(url)
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
// Or use the callback-based API:
//
//	err := sse.Stream(resp.Body, func(event sse.Event) error {
//	    fmt.Println(event.Data)
//	    return nil
//	})
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
type Event struct {
	// ID is the event ID (optional).
	ID string

	// Event is the event type (optional, defaults to "message").
	Event string

	// Data is the event payload.
	Data string

	// Retry is the reconnection time in milliseconds (optional).
	Retry int
}

// JSON unmarshals the event data as JSON into v.
func (e *Event) JSON(v any) error {
	return json.Unmarshal([]byte(e.Data), v)
}

// IsEmpty returns true if the event has no data.
func (e *Event) IsEmpty() bool {
	return e.Data == "" && e.Event == "" && e.ID == ""
}

// Reader reads SSE events from an io.Reader.
type Reader struct {
	scanner *bufio.Scanner
	event   Event
}

// NewReader creates a new SSE reader.
func NewReader(r io.Reader) *Reader {
	return &Reader{
		scanner: bufio.NewScanner(r),
	}
}

// Buffer sets the buffer size for reading lines. The default max line size
// is 64KB. Use this if you need to read events with very long lines.
// Must be called before the first call to Read.
func (r *Reader) Buffer(maxLineSize int) {
	r.scanner.Buffer(make([]byte, 0, maxLineSize), maxLineSize)
}

// Read reads the next event from the stream.
// Returns io.EOF when the stream ends.
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
// Stops on error or when the stream ends.
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

// Client is an SSE client that handles HTTP connections.
type Client struct {
	// URL is the SSE endpoint.
	URL string

	// Headers are additional headers to send with requests.
	Headers http.Header

	// HTTPClient is the HTTP client to use. If nil, http.DefaultClient is used.
	HTTPClient *http.Client

	// LastEventID is sent as Last-Event-ID header if set.
	LastEventID string
}

// NewClient creates a new SSE client for the given URL.
func NewClient(url string) *Client {
	return &Client{
		URL:     url,
		Headers: make(http.Header),
	}
}

// Connect establishes a connection and returns an event channel.
// The channel is closed when the connection ends or ctx is cancelled.
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

// HTTPError represents an HTTP error response.
type HTTPError struct {
	StatusCode int
	Status     string
}

func (e *HTTPError) Error() string {
	return "sse: " + e.Status
}

// ParseBytes parses SSE events from a byte slice.
// Useful for testing or parsing buffered data.
func ParseBytes(data []byte) ([]Event, error) {
	var events []Event
	err := Stream(bytes.NewReader(data), func(event Event) error {
		events = append(events, event)
		return nil
	})
	return events, err
}

// ParseString parses SSE events from a string.
func ParseString(data string) ([]Event, error) {
	return ParseBytes([]byte(data))
}
