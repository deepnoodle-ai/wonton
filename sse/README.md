# sse

Server-Sent Events (SSE) parser and client for streaming data from HTTP endpoints. Commonly used for streaming responses from LLMs like OpenAI, Anthropic Claude, and other AI services.

## Features

- Standards-compliant SSE parsing (handles both LF and CRLF line endings)
- Streaming and callback-based APIs
- Last-Event-ID tracking for manual reconnection
- Configurable buffer sizes for large events
- JSON unmarshaling helpers
- HTTP client with context support and cancellation

## Usage Examples

### Basic Streaming

```go
package main

import (
    "fmt"
    "io"
    "net/http"
    "github.com/deepnoodle-ai/wonton/sse"
)

func main() {
    resp, err := http.Get("https://example.com/events")
    if err != nil {
        panic(err)
    }
    defer resp.Body.Close()

    reader := sse.NewReader(resp.Body)

    for {
        event, err := reader.Read()
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }

        fmt.Printf("Event: %s\nData: %s\n\n", event.Event, event.Data)
    }
}
```

### Callback-Based API

```go
resp, _ := http.Get("https://example.com/events")
defer resp.Body.Close()

err := sse.Stream(resp.Body, func(event sse.Event) error {
    fmt.Println(event.Data)

    // Return error to stop streaming
    if event.Data == "done" {
        return io.EOF
    }

    return nil
})

if err != nil && err != io.EOF {
    log.Fatal(err)
}
```

### JSON Event Data

```go
type Message struct {
    ID      int    `json:"id"`
    Content string `json:"content"`
}

reader := sse.NewReader(resp.Body)

for {
    event, err := reader.Read()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }

    var msg Message
    if err := event.JSON(&msg); err != nil {
        log.Printf("Invalid JSON: %v", err)
        continue
    }

    fmt.Printf("Message %d: %s\n", msg.ID, msg.Content)
}
```

### SSE Client

```go
import "context"

client := sse.NewClient("https://example.com/events")
client.Headers.Set("Authorization", "Bearer token123")

ctx := context.Background()
events, errs := client.Connect(ctx)

for event := range events {
    fmt.Printf("Received: %s\n", event.Data)
}

if err := <-errs; err != nil {
    log.Printf("Error: %v\n", err)
}
```

### Reconnection with Last-Event-ID

The client tracks the last event ID automatically. To reconnect after a
disconnection, call Connect again - the Last-Event-ID header will be sent:

```go
client := sse.NewClient("https://example.com/events")

for {
    events, errs := client.Connect(ctx)

    for event := range events {
        fmt.Printf("Received: %s\n", event.Data)
    }

    if err := <-errs; err != nil {
        if ctx.Err() != nil {
            return // Context cancelled, stop reconnecting
        }
        log.Printf("Disconnected: %v, reconnecting...\n", err)
        time.Sleep(3 * time.Second) // Backoff before reconnecting
        continue
    }
    break // Clean EOF
}
```

### Custom Headers

```go
client := sse.NewClient("https://api.example.com/stream")
client.Headers.Set("Authorization", "Bearer sk-...")
client.Headers.Set("X-Request-ID", "req-123")

ctx := context.Background()
events, errs := client.Connect(ctx)
```

### Large Event Support

```go
// Using Reader directly
reader := sse.NewReader(resp.Body)
reader.Buffer(1024 * 1024) // 1MB max line size

for {
    event, err := reader.Read()
    // ...
}

// Using Client
client := sse.NewClient("https://example.com/events")
client.BufferSize = 1024 * 1024 // 1MB max line size
events, errs := client.Connect(ctx)
```

### Last-Event-ID Support

```go
client := sse.NewClient("https://example.com/events")
client.LastEventID = "1234" // Resume from this event

events, errs := client.Connect(context.Background())

for event := range events {
    // Client automatically updates LastEventID
    fmt.Printf("Event ID: %s\n", event.ID)
}
```

### Parse String/Bytes

```go
// Useful for testing or processing buffered SSE data
data := `event: message
data: Hello

event: message
data: World

`

events, err := sse.ParseString(data)
if err != nil {
    panic(err)
}

for _, event := range events {
    fmt.Println(event.Data)
}
// Output:
// Hello
// World
```

### OpenAI Streaming Example

```go
import (
    "encoding/json"
    "net/http"
)

func streamOpenAI(prompt string) error {
    req, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions",
        strings.NewReader(`{"model":"gpt-4","messages":[{"role":"user","content":"`+prompt+`"}],"stream":true}`))
    req.Header.Set("Authorization", "Bearer "+apiKey)
    req.Header.Set("Content-Type", "application/json")

    resp, err := http.DefaultClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    return sse.Stream(resp.Body, func(event sse.Event) error {
        if event.Data == "[DONE]" {
            return io.EOF
        }

        var chunk struct {
            Choices []struct {
                Delta struct {
                    Content string `json:"content"`
                } `json:"delta"`
            } `json:"choices"`
        }

        if err := event.JSON(&chunk); err != nil {
            return err
        }

        if len(chunk.Choices) > 0 {
            fmt.Print(chunk.Choices[0].Delta.Content)
        }

        return nil
    })
}
```

## API Reference

### Functions

| Function | Description | Inputs | Outputs |
|----------|-------------|--------|---------|
| `NewReader` | Creates new SSE reader | `r io.Reader` | `*Reader` |
| `Stream` | Reads all events and calls function for each | `r io.Reader, fn func(Event) error` | `error` |
| `NewClient` | Creates new SSE client | `url string` | `*Client` |
| `ParseBytes` | Parses SSE events from byte slice | `data []byte` | `[]Event, error` |
| `ParseString` | Parses SSE events from string | `data string` | `[]Event, error` |

### Types

#### Event

Represents a single Server-Sent Event.

```go
type Event struct {
    ID    string  // Event ID (optional)
    Event string  // Event type (optional, defaults to "message")
    Data  string  // Event payload
    Retry int     // Reconnection time in milliseconds (optional)
}
```

Methods:
- `JSON(v any) error` - Unmarshal event data as JSON
- `IsEmpty() bool` - Returns true if event has no data

#### Reader

SSE event reader.

```go
type Reader struct {
    // private fields
}
```

Methods:
- `Read() (Event, error)` - Read next event (returns io.EOF when stream ends)
- `Buffer(maxLineSize int)` - Set buffer size for reading lines

#### Client

HTTP SSE client with Last-Event-ID tracking.

```go
type Client struct {
    URL         string       // SSE endpoint URL
    Headers     http.Header  // Additional headers to send
    HTTPClient  *http.Client // HTTP client (nil = http.DefaultClient)
    LastEventID string       // Sent as Last-Event-ID header
    BufferSize  int          // Max line size (0 = 64KB default)
}
```

Methods:
- `Connect(ctx context.Context) (<-chan Event, <-chan error)` - Establish connection and return event/error channels

Note: Connect returns when the stream ends or an error occurs. For reconnection,
call Connect again in a loop - LastEventID will be sent automatically.

#### HTTPError

HTTP error response.

```go
type HTTPError struct {
    StatusCode int
    Status     string
}
```

## SSE Format

Server-Sent Events follow this format:

```
event: message
data: First line
data: Second line
id: 123
retry: 5000

```

- Lines starting with `:` are comments (ignored)
- Empty line marks end of event
- Multiple `data:` lines are joined with newlines
- The `id:` field sets the event ID
- The `retry:` field sets reconnection delay in milliseconds
- The `event:` field sets the event type (default: "message")

## Related Packages

- [fetch](../fetch) - HTTP fetching with configurable options
- [retry](../retry) - Add retry logic to SSE connections
- [htmltomd](../htmltomd) - Convert streamed HTML to Markdown

## Design Notes

The package supports both pull-based (`Read()` in a loop) and push-based (`Stream()` with callback) APIs. The pull-based API gives more control, while the push-based API is more convenient for simple use cases.

The Client type handles HTTP connection setup and sends appropriate SSE headers. It automatically tracks and sends the Last-Event-ID header for reconnection scenarios. Client does not automatically reconnect - implement reconnection by calling Connect in a loop.

The reader has a default maximum line size of 64KB. Use `Buffer()` to increase this if you expect very long event data lines, or set `Client.BufferSize` when using the client. The buffer size must be set before the first call to `Read()`.

**Channel behavior**: `Client.Connect()` returns an unbuffered event channel. The sending goroutine will block if the consumer stops reading events. Always cancel the context when done to ensure the goroutine exits promptly. The send operation selects on both the channel and context, so proper context cancellation prevents goroutine leaks.

**Spec compliance**: Events are only emitted when they contain data (per SSE spec). The event type defaults to "message" if not specified. Both LF and CRLF line endings are supported. The last event ID persists across events that have no data.
