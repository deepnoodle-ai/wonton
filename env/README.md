# env configuration helpers

The `env` package loads strongly typed configuration structs from environment
variables, `.env` files, and JSON blobs. It supports stage-specific overrides,
default values, custom parsers, and aggregate error reporting.

## Features

- Declarative struct tags: `env:"PORT" default:"8080" required` style definitions.
- Stage and prefix support: resolve `PROD_PORT` before `PORT`, or automatically
  prepend `MYAPP_`.
- File inputs: merge multiple `.env` and JSON files with later files overriding
  earlier ones.
- Custom parsers via `env.WithParser`, change notification via `env.WithOnSet`.
- Aggregate errors that tell you exactly which fields failed to parse.

## Example

```go
package main

import (
	"log"
	"time"

	"github.com/deepnoodle-ai/wonton/env"
)

type Config struct {
	Host     string        `env:"HOST" default:"127.0.0.1"`
	Port     int           `env:"PORT" default:"8080"`
	APIKey   string        `env:"API_KEY,required"`
	Timeout  time.Duration `env:"TIMEOUT" default:"5s"`
	LogLevel string        `env:"LOG_LEVEL" default:"info"`
}

func main() {
	cfg, err := env.Parse[Config](
		env.WithPrefix("WONTON"),
		env.WithStage("PROD"),
		env.WithEnvFile(".env", ".env.local"),
		env.WithJSONFile("config.json"),
		env.WithRequiredIfNoDefault(),
	)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Serving %s:%d (log=%s)", cfg.Host, cfg.Port, cfg.LogLevel)
}
```

Use `env.ParseInto(&cfg, ...)` to populate an existing struct or inject a custom
environment map with `env.WithEnvironment`. See `examples/env_config` for a
CLI-driven walkthrough that shows error handling and stage overrides.
