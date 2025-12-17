# env

The env package loads strongly-typed configuration structs from environment variables, .env files, and JSON files. It supports stage-specific overrides, default values, custom parsers, nested structs, slices, maps, and aggregate error reporting.

## Usage Examples

### Basic Configuration

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/deepnoodle-ai/wonton/env"
)

type Config struct {
    Host    string        `env:"HOST" envDefault:"localhost"`
    Port    int           `env:"PORT" envDefault:"8080"`
    APIKey  string        `env:"API_KEY,required"`
    Debug   bool          `env:"DEBUG" envDefault:"false"`
    Timeout time.Duration `env:"TIMEOUT" envDefault:"30s"`
}

func main() {
    cfg, err := env.Parse[Config]()
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("Server: %s:%d\n", cfg.Host, cfg.Port)
    fmt.Printf("Timeout: %v\n", cfg.Timeout)
}
```

### Loading from .env Files

```go
cfg, err := env.Parse[Config](
    env.WithEnvFile(".env", ".env.local"),
)
if err != nil {
    log.Fatal(err)
}
```

### Stage-Based Configuration

```go
// With stage "PROD", looks for PROD_PORT before PORT
cfg, err := env.Parse[Config](
    env.WithStage("PROD"),
    env.WithEnvFile(".env"),
)
```

### Prefix Support

```go
// All vars prefixed with MYAPP_ (e.g., MYAPP_HOST, MYAPP_PORT)
cfg, err := env.Parse[Config](
    env.WithPrefix("MYAPP"),
)
```

### JSON Configuration Files

```go
type Config struct {
    Host string `json:"host" env:"HOST" envDefault:"localhost"`
    Port int    `json:"port" env:"PORT" envDefault:"8080"`
}

cfg, err := env.Parse[Config](
    env.WithJSONFile("config.json", "config.local.json"),
)
// Environment variables override JSON values
```

### Nested Structs

```go
type Database struct {
    Host     string `env:"HOST" envDefault:"localhost"`
    Port     int    `env:"PORT" envDefault:"5432"`
    Username string `env:"USER,required"`
    Password string `env:"PASSWORD,required,unset"`
}

type Config struct {
    DB Database `envPrefix:"DB_"`
}

// Expects: DB_HOST, DB_PORT, DB_USER, DB_PASSWORD
cfg, err := env.Parse[Config]()
```

### Slices and Maps

```go
type Config struct {
    // Comma-separated: "host1,host2,host3"
    Hosts []string `env:"HOSTS" envSeparator:","`

    // Colon-separated key:value pairs: "key1:val1,key2:val2"
    Labels map[string]string `env:"LABELS"`

    // Custom separators
    Ports []int `env:"PORTS" envSeparator:";"`
}
```

### Custom Parsers

```go
type IPAddr struct{ net.IP }

cfg, err := env.Parse[Config](
    env.WithParser(func(s string) (IPAddr, error) {
        ip := net.ParseIP(s)
        if ip == nil {
            return IPAddr{}, fmt.Errorf("invalid IP: %s", s)
        }
        return IPAddr{ip}, nil
    }),
)
```

### Field Change Notifications

```go
cfg, err := env.Parse[Config](
    env.WithOnSet(func(fieldName, envVar string, value any, isDefault bool) {
        if isDefault {
            fmt.Printf("%s: using default value\n", fieldName)
        } else {
            fmt.Printf("%s: loaded from %s\n", fieldName, envVar)
        }
    }),
)
```

### Loading File Contents

```go
type Config struct {
    // Reads the file path from API_KEY_FILE, then loads file contents
    APIKey string `env:"API_KEY_FILE,file,required"`
}
```

### Variable Expansion

```go
type Config struct {
    // Expands $HOME and other env vars in the value
    LogPath string `env:"LOG_PATH,expand" envDefault:"$HOME/logs/app.log"`
}
```

## API Reference

### Configuration Functions

| Function | Description | Returns |
|----------|-------------|---------|
| `Parse[T](opts...)` | Parses environment into type T | `(T, error)` |
| `Must[T](opts...)` | Like Parse but panics on error | `T` |
| `ParseInto(v any, opts...)` | Parses into existing struct pointer | `error` |

### Options

| Function | Description |
|----------|-------------|
| `WithPrefix(prefix)` | Prepends prefix to all env var names |
| `WithStage(stage)` | Enables stage-based variable resolution |
| `WithEnvFile(files...)` | Loads .env files in order |
| `WithJSONFile(files...)` | Loads JSON config files in order |
| `WithEnvironment(map)` | Uses custom env map instead of os.Environ() |
| `WithTagName(env, default)` | Changes struct tag names |
| `WithRequiredIfNoDefault()` | Makes fields without defaults required |
| `WithUseFieldName()` | Uses field names as env vars when no tag present |
| `WithParser[T](parser)` | Adds custom parser for type T |
| `WithOnSet(fn)` | Calls fn when fields are set |
| `WithRequireConfigFile()` | Errors if no .env or JSON file loaded |

### Struct Tags

| Tag | Description | Example |
|-----|-------------|---------|
| `env:"VAR"` | Environment variable name | `env:"PORT"` |
| `env:"VAR,required"` | Field must be set | `env:"API_KEY,required"` |
| `env:"VAR,notEmpty"` | Field must be set and non-empty | `env:"NAME,notEmpty"` |
| `env:"VAR,file"` | Value is file path; read file contents | `env:"KEY_FILE,file"` |
| `env:"VAR,expand"` | Expand $VAR references in value | `env:"PATH,expand"` |
| `env:"VAR,unset"` | Unset env var after reading (for secrets) | `env:"PASSWORD,unset"` |
| `envDefault:"value"` | Default value if not set | `envDefault:"8080"` |
| `envPrefix:"PREFIX_"` | Prefix for nested struct fields | `envPrefix:"DB_"` |
| `envSeparator:","` | Separator for slice/map parsing | `envSeparator:";"` |
| `envKeyValSeparator:":"` | Separator for map key:value pairs | `envKeyValSeparator:"="` |

### .env File Functions

| Function | Description | Returns |
|----------|-------------|---------|
| `LoadEnvFile(files...)` | Loads .env files into os environment | `error` |
| `OverloadEnvFile(files...)` | Loads .env files, overwriting existing vars | `error` |
| `ReadEnvFile(filename)` | Reads .env file into map | `(map[string]string, error)` |
| `ParseEnvReader(r)` | Parses .env format from reader | `(map[string]string, error)` |
| `ParseEnvString(s)` | Parses .env format string | `(map[string]string, error)` |
| `WriteEnvFile(map, filename)` | Writes map to .env file (default perms) | `error` |
| `WriteEnvFileWithPerm(map, filename, perm)` | Writes map to .env file with permissions | `error` |

### Supported Types

- Basic types: `string`, `bool`, `int`, `int8`, `int16`, `int32`, `int64`, `uint`, `uint8`, `uint16`, `uint32`, `uint64`, `float32`, `float64`
- Time types: `time.Duration`, `time.Location`
- Collections: `[]T` (slices), `map[K]V` (maps)
- Any type implementing `encoding.TextUnmarshaler`
- Pointer types: `*T` for any supported type T

### Error Types

| Type | Description |
|------|-------------|
| `ParseError` | General parsing error |
| `AggregateError` | Multiple field errors collected together |
| `FieldError` | Error parsing a specific field |
| `VarNotSetError` | Required variable not set |
| `EmptyVarError` | Variable set but empty when notEmpty required |
| `FileLoadError` | Error loading file when file option used |

## .env File Format

The package supports standard .env file syntax:

```bash
# Comments start with # or //
HOST=localhost
PORT=8080

# Quoted values
DATABASE_URL="postgres://localhost/mydb"

# Variable expansion (with expand tag)
LOG_PATH=$HOME/logs/app.log

# Export syntax
export API_KEY=secret123

# Empty values
OPTIONAL_FIELD=

# Newlines via escape sequences (double-quoted only)
PRIVATE_KEY="-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...\n-----END RSA PRIVATE KEY-----"
```

## Related Packages

- [cli](../cli/) - CLI framework that uses env for configuration loading
- [terminal](../terminal/) - Terminal utilities for interactive configuration

## Implementation Notes

### Configuration Precedence (lowest to highest)

1. **envDefault tags** - Default values specified in struct tags
2. **JSON files** - Unmarshaled directly into struct (later files override earlier ones)
3. **.env files** - Merged together (later files override earlier ones)
4. **Environment variables** - Process env vars (or custom map via `WithEnvironment`)

This means environment variables always win, followed by .env files, then JSON, then defaults.

### JSON and Zero Values

When using JSON files with `envDefault`, explicit zero values in JSON (0, false, "") cannot be reliably distinguished from "not set". The parser treats zero values as "not set" and applies the default. To work around this:

- Use pointer types (`*int`, `*bool`) where nil means "not set" and zero is explicit
- Avoid `envDefault` for fields where zero is a meaningful value from JSON
- Use environment variables to override JSON when you need explicit zeros

### Other Notes

- Missing config files are silently skipped unless `WithRequireConfigFile()` is used
- Nested structs can use `envPrefix` tag or auto-generate PREFIX_FIELDNAME_ format
- Aggregate errors provide detailed information about all parsing failures
