package sse

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/deepnoodle-ai/wonton/assert"
)

func streamServer(t *testing.T, body string, gotLastID *string) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if gotLastID != nil {
			*gotLastID = r.Header.Get("Last-Event-ID")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(server.Close)
	return server
}

func drain(t *testing.T, client *Client) []Event {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	events, errs := client.Connect(ctx)
	var got []Event
	for event := range events {
		got = append(got, event)
	}
	// LastEventID is read only after the events channel closes, as its
	// documentation requires. Run with -race.
	assert.NoError(t, <-errs)
	return got
}

// BUG-062: the client must not race on LastEventID.
func TestClientLastEventIDNoRace(t *testing.T) {
	server := streamServer(t, "id: 1\ndata: a\n\nid: 2\ndata: b\n\nid: 3\ndata: c\n\n", nil)
	client := NewClient(server.URL)
	got := drain(t, client)
	assert.Equal(t, 3, len(got))
	// While the stream runs, Event.ID carries the last event ID.
	assert.Equal(t, "2", got[1].ID)
	assert.Equal(t, "3", client.LastEventID)
}

// Reading LastEventID right after the final error must not race with the
// client goroutine either.
func TestClientLastEventIDAfterError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("id: 7\ndata: a\n\n"))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	t.Cleanup(server.Close)
	client := NewClient(server.URL)
	ctx, cancel := context.WithCancel(context.Background())
	events, errs := client.Connect(ctx)
	<-events
	cancel()
	assert.Error(t, <-errs)
	assert.Equal(t, "7", client.LastEventID)
	for range events {
	}
}

// BUG-062: an empty id field resets the last event ID, and an id with no
// data still updates it.
func TestClientLastEventIDResetAndIDOnly(t *testing.T) {
	server := streamServer(t, "id: 1\ndata: a\n\nid\ndata: b\n\n", nil)
	client := NewClient(server.URL)
	drain(t, client)
	assert.Equal(t, "", client.LastEventID)

	server = streamServer(t, "id: 1\ndata: a\n\nid: 2\n\n", nil)
	client = NewClient(server.URL)
	drain(t, client)
	assert.Equal(t, "2", client.LastEventID)
}

// BUG-062: on reconnect, events without an id carry the previous ID.
func TestClientSeedsReaderWithLastEventID(t *testing.T) {
	var sent string
	server := streamServer(t, "data: a\n\n", &sent)
	client := NewClient(server.URL)
	client.LastEventID = "5"
	got := drain(t, client)
	assert.Equal(t, "5", sent)
	assert.Equal(t, 1, len(got))
	assert.Equal(t, "5", got[0].ID)
	assert.Equal(t, "5", client.LastEventID)
}

// BUG-062: a lone CR is a line ending.
func TestReaderCROnlyLineEndings(t *testing.T) {
	events, err := ParseString("event: ping\rdata: hello\r\rdata: a\r\ndata: b\r\n\r\n")
	assert.NoError(t, err)
	assert.Equal(t, 2, len(events))
	assert.Equal(t, "ping", events[0].Event)
	assert.Equal(t, "hello", events[0].Data)
	assert.Equal(t, "a\nb", events[1].Data)
}

func TestReaderCRDoesNotWaitForNextByte(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()
	go func() { _, _ = pw.Write([]byte("data: x\r\r")) }()

	reader := NewReader(pr)
	done := make(chan Event, 1)
	go func() {
		event, _ := reader.Read()
		done <- event
	}()
	select {
	case event := <-done:
		assert.Equal(t, "x", event.Data)
	case <-time.After(2 * time.Second):
		t.Fatal("event ending in CR was not delivered until more input arrived")
	}

	// A CRLF split across reads is still one line ending.
	go func() { _, _ = pw.Write([]byte("data: y\r")) }()
	go func() {
		time.Sleep(10 * time.Millisecond)
		_, _ = pw.Write([]byte("\ndata: z\n\n"))
	}()
	event, err := reader.Read()
	assert.NoError(t, err)
	assert.Equal(t, "y\nz", event.Data)
}
