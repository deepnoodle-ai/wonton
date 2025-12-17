package sse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func TestReaderBasic(t *testing.T) {
	data := `event: message
data: hello world

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	assert.NoError(t, err)
	assert.Equal(t, "message", event.Event)
	assert.Equal(t, "hello world", event.Data)

	_, err = reader.Read()
	assert.Equal(t, io.EOF, err)
}

func TestReaderMultipleEvents(t *testing.T) {
	data := `data: first

data: second

data: third

`
	reader := NewReader(strings.NewReader(data))

	events := []string{}
	for {
		event, err := reader.Read()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)
		events = append(events, event.Data)
	}

	assert.Equal(t, []string{"first", "second", "third"}, events)
}

func TestReaderMultilineData(t *testing.T) {
	data := `data: line 1
data: line 2
data: line 3

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	assert.NoError(t, err)
	assert.Equal(t, "line 1\nline 2\nline 3", event.Data)
}

func TestReaderWithID(t *testing.T) {
	data := `id: 123
event: update
data: payload

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	assert.NoError(t, err)
	assert.Equal(t, "123", event.ID)
	assert.Equal(t, "update", event.Event)
	assert.Equal(t, "payload", event.Data)
}

func TestReaderWithRetry(t *testing.T) {
	data := `retry: 5000
data: reconnect info

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	assert.NoError(t, err)
	assert.Equal(t, 5000, event.Retry)
	assert.Equal(t, "reconnect info", event.Data)
}

func TestReaderComments(t *testing.T) {
	data := `: this is a comment
data: actual data
: another comment

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	assert.NoError(t, err)
	assert.Equal(t, "actual data", event.Data)
}

func TestReaderNoSpace(t *testing.T) {
	// SSE spec says space after colon is optional
	data := `data:no-space

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	assert.NoError(t, err)
	assert.Equal(t, "no-space", event.Data)
}

func TestEventJSON(t *testing.T) {
	event := Event{
		Data: `{"message": "hello", "count": 42}`,
	}

	var result struct {
		Message string `json:"message"`
		Count   int    `json:"count"`
	}

	err := event.JSON(&result)
	assert.NoError(t, err)
	assert.Equal(t, "hello", result.Message)
	assert.Equal(t, 42, result.Count)
}

func TestStream(t *testing.T) {
	data := `data: one

data: two

data: three

`
	var events []string
	err := Stream(strings.NewReader(data), func(event Event) error {
		events = append(events, event.Data)
		return nil
	})

	assert.NoError(t, err)
	assert.Equal(t, []string{"one", "two", "three"}, events)
}

func TestParseString(t *testing.T) {
	data := `event: ping
data: {}

event: message
data: {"text": "hello"}

`
	events, err := ParseString(data)
	assert.NoError(t, err)
	assert.Len(t, events, 2)
	assert.Equal(t, "ping", events[0].Event)
	assert.Equal(t, "message", events[1].Event)
}

func TestClientConnect(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "text/event-stream", r.Header.Get("Accept"))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher := w.(http.Flusher)

		w.Write([]byte("data: event 1\n\n"))
		flusher.Flush()

		w.Write([]byte("data: event 2\n\n"))
		flusher.Flush()
	}))
	defer server.Close()

	client := NewClient(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	events, errs := client.Connect(ctx)

	var received []string
	for event := range events {
		received = append(received, event.Data)
	}

	// Check for errors
	select {
	case err := <-errs:
		if err != nil && err != context.DeadlineExceeded {
			t.Fatalf("unexpected error: %v", err)
		}
	default:
	}

	assert.Equal(t, []string{"event 1", "event 2"}, received)
}

func TestClientLastEventID(t *testing.T) {
	var receivedLastEventID string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedLastEventID = r.Header.Get("Last-Event-ID")

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("id: 456\ndata: response\n\n"))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	client.LastEventID = "123"

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	events, _ := client.Connect(ctx)
	for range events {
	}

	assert.Equal(t, "123", receivedLastEventID)
	assert.Equal(t, "456", client.LastEventID)
}

func TestHTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	events, errs := client.Connect(ctx)

	// Drain events channel
	for range events {
	}

	err := <-errs
	assert.Error(t, err)

	var httpErr *HTTPError
	assert.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusUnauthorized, httpErr.StatusCode)
}

func TestEventIsEmpty(t *testing.T) {
	empty := Event{}
	assert.True(t, empty.IsEmpty())

	withData := Event{Data: "something"}
	assert.False(t, withData.IsEmpty())

	withEvent := Event{Event: "ping"}
	assert.False(t, withEvent.IsEmpty())
}

func TestReaderBuffer(t *testing.T) {
	// Create a line longer than the default 64KB
	longData := strings.Repeat("x", 100000)
	data := "data: " + longData + "\n\n"

	reader := NewReader(strings.NewReader(data))
	reader.Buffer(200000) // Set buffer large enough

	event, err := reader.Read()
	assert.NoError(t, err)
	assert.Equal(t, longData, event.Data)
}

func TestContentTypeValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("<html>not sse</html>"))
	}))
	defer server.Close()

	client := NewClient(server.URL)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	events, errs := client.Connect(ctx)

	// Drain events channel
	for range events {
	}

	err := <-errs
	assert.Error(t, err)

	var httpErr *HTTPError
	assert.ErrorAs(t, err, &httpErr)
	assert.Contains(t, httpErr.Status, "unexpected content-type")
}
