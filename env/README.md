# env configuration helpers

The `env` package loads strongly typed configuration structs from environment
variables, `.env` files, and JSON blobs. It supports stage-specific overrides,
default values, custom parsers, and aggregate error reporting.

## Features

- Declarative struct tags: `env:"PORT" envDefault:"8080"` or `env:"PORT,required"` style definitions.
- Stage and prefix support: resolve `PROD_PORT` before `PORT`, or automatically
  prepend `MYAPP_`.
- File inputs: merge multiple `.env` and JSON files with later files overriding
  earlier ones.
- Custom parsers via `env.WithParser`, change notification via `env.WithOnSet`.
- Aggregate errors that tell you exactly which fields failed to parse.

## Tag Options

| Tag                      | Description                                       |
| ------------------------ | ------------------------------------------------- |
| `env:"VAR"`              | Environment variable name                         |
| `env:"VAR,required"`     | Field must be set                                 |
| `env:"VAR,notEmpty"`     | Field must be set and non-empty                   |
| `env:"VAR,file"`         | Value is a file path; read contents as value      |
| `env:"VAR,expand"`       | Expand `$VAR` references in value                 |
| `env:"VAR,unset"`        | Unset env var after reading (for secrets)         |
| `envDefault:"value"`     | Default value if not set (also supports `default`) |
| `envSeparator:","`       | Custom separator for slices (default: `,`)        |
| `envKeyValSeparator:":"` | Custom separator for map key:value (default: `:`) |

## Built-in Type Support

- All basic types: `string`, `bool`, `int*`, `uint*`, `float*`
- `time.Duration` (e.g., `"5s"`, `"1h30m"`)
- `time.Location` (e.g., `"America/New_York"`, `"UTC"`)
- Slices and maps with customizable separators
- Any type implementing `encoding.TextUnmarshaler`

## JSON Config Files

JSON files are loaded using standard `json.Unmarshal`, so use `json` tags to
control field mapping. Environment variables override JSON values.

```go
type Config struct {
    Host string `json:"host" env:"HOST" envDefault:"localhost"`
    Port int    `json:"port" env:"PORT" envDefault:"8080"`
}
```

```json
{ "host": "example.com", "port": 3000 }
```

## Example

```go
type Config struct {
	Host    string        `env:"HOST" envDefault:"localhost"`
	Port    int           `env:"PORT" envDefault:"8080"`
	APIKey  string        `env:"API_KEY,required"`
	Timeout time.Duration `env:"TIMEOUT" envDefault:"30s"`
}

cfg, err := env.Parse[Config](
	env.WithEnvFile(".env"),
)
```
