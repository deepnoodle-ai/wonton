# Server-Sent Events helpers

The `sse` package parses Server-Sent Events streams (like the ones returned by
LLM APIs) and provides both a low-level reader and a high-level reconnecting
client.

## Usage

```go
package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/deepnoodle-ai/wonton/sse"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	client := sse.NewClient("https://example.com/stream")
	client.Headers.Set("Accept", "text/event-stream")

	events, errs := client.Connect(ctx)
	for {
		select {
		case event, ok := <-events:
			if !ok {
				return
			}
			log.Printf("[%s] %s", event.Event, event.Data)
		case err := <-errs:
			if err == nil || err == io.EOF {
				return
			}
			log.Fatal(err)
		}
	}
}
```

For one-off streams where you already have an `io.Reader`, use `sse.Reader` or
`sse.Stream`:

```go
resp, _ := http.Get("https://example.com/stream")
defer resp.Body.Close()

if err := sse.Stream(resp.Body, func(ev sse.Event) error {
	var payload map[string]any
	_ = ev.JSON(&payload)
	log.Println(payload)
	return nil
}); err != nil {
	log.Fatal(err)
}
```

Each `Event` exposes helper methods like `JSON` (for decoding the data field) and
`IsEmpty`. The client automatically sends `Last-Event-ID` headers when the server
supports resumable streams.
