# clipboard

Cross-platform system clipboard access for Go. Read from and write to the system clipboard on macOS, Linux, and Windows.

## Summary

The clipboard package provides a simple, context-aware API for interacting with the system clipboard across all major platforms. It uses native clipboard utilities (pbcopy/pbpaste on macOS, xclip/xsel/wl-copy/wl-paste on Linux, PowerShell on Windows) and includes timeout protection, context cancellation support, and availability checking. All operations are safe for concurrent use and support custom timeouts.

## Usage Examples

### Basic Read and Write

```go
package main

import (
    "fmt"
    "log"

    "github.com/deepnoodle-ai/wonton/clipboard"
)

func main() {
    // Write to clipboard
    err := clipboard.Write("Hello, World!")
    if err != nil {
        log.Fatal(err)
    }

    // Read from clipboard
    text, err := clipboard.Read()
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Clipboard contents:", text)
}
```

### With Context Support

```go
package main

import (
    "context"
    "time"

    "github.com/deepnoodle-ai/wonton/clipboard"
)

func main() {
    ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer cancel()

    // Write with context
    err := clipboard.WriteContext(ctx, "Important data")
    if err != nil {
        log.Fatal(err)
    }

    // Read with context
    text, err := clipboard.ReadContext(ctx)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(text)
}
```

### Custom Timeouts

```go
package main

import (
    "time"

    "github.com/deepnoodle-ai/wonton/clipboard"
)

func main() {
    // Use a shorter timeout for clipboard operations
    timeout := 1 * time.Second

    err := clipboard.WriteWithTimeout("Fast write", timeout)
    if err == clipboard.ErrTimeout {
        fmt.Println("Write operation timed out")
        return
    }

    text, err := clipboard.ReadWithTimeout(timeout)
    if err == clipboard.ErrTimeout {
        fmt.Println("Read operation timed out")
        return
    }
    fmt.Println(text)
}
```

### Availability Checking

```go
package main

import (
    "fmt"

    "github.com/deepnoodle-ai/wonton/clipboard"
)

func main() {
    if !clipboard.Available() {
        fmt.Println("Clipboard not available on this system")
        return
    }

    // Safe to use clipboard operations
    clipboard.Write("Data")
}
```

### Clear Clipboard

```go
package main

import (
    "github.com/deepnoodle-ai/wonton/clipboard"
)

func main() {
    // Write some data
    clipboard.Write("Secret data")

    // Clear the clipboard when done
    err := clipboard.Clear()
    if err != nil {
        log.Fatal(err)
    }
}
```

### Error Handling

```go
package main

import (
    "errors"
    "fmt"

    "github.com/deepnoodle-ai/wonton/clipboard"
)

func main() {
    text, err := clipboard.Read()
    if err != nil {
        switch {
        case errors.Is(err, clipboard.ErrUnavailable):
            fmt.Println("Clipboard not available on this system")
        case errors.Is(err, clipboard.ErrTimeout):
            fmt.Println("Clipboard operation timed out")
        default:
            fmt.Printf("Error: %v\n", err)
        }
        return
    }
    fmt.Println(text)
}
```

### CLI Tool Example

```go
package main

import (
    "fmt"
    "io"
    "os"

    "github.com/deepnoodle-ai/wonton/clipboard"
)

func main() {
    if len(os.Args) > 1 && os.Args[1] == "paste" {
        // Read from clipboard and print to stdout
        text, err := clipboard.Read()
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        fmt.Print(text)
    } else {
        // Read from stdin and write to clipboard
        data, err := io.ReadAll(os.Stdin)
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
        err = clipboard.Write(string(data))
        if err != nil {
            fmt.Fprintf(os.Stderr, "Error: %v\n", err)
            os.Exit(1)
        }
    }
}
```

## API Reference

### Read Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `Read()` | Reads text from clipboard with default timeout (5s) | None | `string`, `error` |
| `ReadWithTimeout(timeout)` | Reads text with custom timeout | `time.Duration` | `string`, `error` |
| `ReadContext(ctx)` | Reads text with context support | `context.Context` | `string`, `error` |

### Write Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `Write(text)` | Writes text to clipboard with default timeout (5s) | `string` | `error` |
| `WriteWithTimeout(text, timeout)` | Writes text with custom timeout | `string`, `time.Duration` | `error` |
| `WriteContext(ctx, text)` | Writes text with context support | `context.Context`, `string` | `error` |

### Utility Functions

| Function | Description | Parameters | Returns |
|----------|-------------|------------|---------|
| `Available()` | Checks if clipboard is available on this system | None | `bool` |
| `Clear()` | Clears clipboard contents (writes empty string) | None | `error` |

### Error Types

| Error | Description |
|-------|-------------|
| `ErrUnavailable` | Clipboard access not available on this system |
| `ErrTimeout` | Clipboard operation exceeded timeout |

### Default Values

| Constant | Value | Description |
|----------|-------|-------------|
| `defaultTimeout` | 5 seconds | Default timeout for all operations |

## Platform Support

### macOS
- Uses `pbcopy` for writing
- Uses `pbpaste` for reading
- Native utilities, always available

### Linux
- **X11**: Uses `xclip` or `xsel` (checks for availability)
- **Wayland**: Uses `wl-copy` and `wl-paste`
- Auto-detects available clipboard tool

### Windows
- Uses PowerShell's `clip.exe` for writing
- Uses PowerShell's `Get-Clipboard` for reading
- Always available (PowerShell is built-in)

## Related Packages

- **[cli](../cli/)** - CLI framework that can use clipboard for copy/paste functionality
- **[terminal](../terminal/)** - Terminal control for building clipboard-aware TUI apps
- **[tui](../tui/)** - Terminal UI library with clipboard integration support
