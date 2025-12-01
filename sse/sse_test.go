package sse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/require"
)

func TestReaderBasic(t *testing.T) {
	data := `event: message
data: hello world

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, "message", event.Event)
	require.Equal(t, "hello world", event.Data)

	_, err = reader.Read()
	require.Equal(t, io.EOF, err)
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
		require.NoError(t, err)
		events = append(events, event.Data)
	}

	require.Equal(t, []string{"first", "second", "third"}, events)
}

func TestReaderMultilineData(t *testing.T) {
	data := `data: line 1
data: line 2
data: line 3

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, "line 1\nline 2\nline 3", event.Data)
}

func TestReaderWithID(t *testing.T) {
	data := `id: 123
event: update
data: payload

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, "123", event.ID)
	require.Equal(t, "update", event.Event)
	require.Equal(t, "payload", event.Data)
}

func TestReaderWithRetry(t *testing.T) {
	data := `retry: 5000
data: reconnect info

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, 5000, event.Retry)
	require.Equal(t, "reconnect info", event.Data)
}

func TestReaderComments(t *testing.T) {
	data := `: this is a comment
data: actual data
: another comment

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, "actual data", event.Data)
}

func TestReaderNoSpace(t *testing.T) {
	// SSE spec says space after colon is optional
	data := `data:no-space

`
	reader := NewReader(strings.NewReader(data))

	event, err := reader.Read()
	require.NoError(t, err)
	require.Equal(t, "no-space", event.Data)
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
	require.NoError(t, err)
	require.Equal(t, "hello", result.Message)
	require.Equal(t, 42, result.Count)
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

	require.NoError(t, err)
	require.Equal(t, []string{"one", "two", "three"}, events)
}

func TestParseString(t *testing.T) {
	data := `event: ping
data: {}

event: message
data: {"text": "hello"}

`
	events, err := ParseString(data)
	require.NoError(t, err)
	require.Len(t, events, 2)
	require.Equal(t, "ping", events[0].Event)
	require.Equal(t, "message", events[1].Event)
}

func TestClientConnect(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "text/event-stream", r.Header.Get("Accept"))

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

	require.Equal(t, []string{"event 1", "event 2"}, received)
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

	require.Equal(t, "123", receivedLastEventID)
	require.Equal(t, "456", client.LastEventID)
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
	require.Error(t, err)

	var httpErr *HTTPError
	require.ErrorAs(t, err, &httpErr)
	require.Equal(t, http.StatusUnauthorized, httpErr.StatusCode)
}

func TestEventIsEmpty(t *testing.T) {
	empty := Event{}
	require.True(t, empty.IsEmpty())

	withData := Event{Data: "something"}
	require.False(t, withData.IsEmpty())

	withEvent := Event{Event: "ping"}
	require.False(t, withEvent.IsEmpty())
}
