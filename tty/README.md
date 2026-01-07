# tty

Package `tty` provides terminal detection utilities.

The primary function is `IsTerminal`, which reports whether a file is connected to a terminal. This is useful for deciding whether to enable interactive features like colors, progress bars, or prompts.

## Usage

```go
import "github.com/deepnoodle-ai/wonton/tty"

if tty.IsTerminal(os.Stdout) {
    // Enable colors, progress bars, etc.
}
```
