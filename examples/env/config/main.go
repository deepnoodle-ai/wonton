// Example demonstrating the env package for configuration loading.
//
// MYAPP_DB_NAME is required, so run with:
//
//	MYAPP_DB_NAME=appdb MYAPP_DB_PASSWORD=secret go run ./examples/env/config
package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/deepnoodle-ai/wonton/env"
)

// Database holds database configuration.
type Database struct {
	Host     string `env:"HOST" envDefault:"localhost"`
	Port     int    `env:"PORT" envDefault:"5432"`
	Name     string `env:"NAME,required"`
	User     string `env:"USER" envDefault:"postgres"`
	Password string `env:"PASSWORD,notEmpty"`
}

// Server holds server configuration.
type Server struct {
	Host         string        `env:"HOST" envDefault:"0.0.0.0"`
	Port         int           `env:"PORT" envDefault:"8080"`
	ReadTimeout  time.Duration `env:"READ_TIMEOUT" envDefault:"30s"`
	WriteTimeout time.Duration `env:"WRITE_TIMEOUT" envDefault:"30s"`
}

// Config is the main application configuration.
type Config struct {
	// Nested structs with prefixes
	Database Database `envPrefix:"DB_"`
	Server   Server   `envPrefix:"SERVER_"`

	// Simple fields
	Debug   bool     `env:"DEBUG" envDefault:"false"`
	LogFile string   `env:"LOG_FILE,file"` // Load file content from path
	Hosts   []string `env:"ALLOWED_HOSTS"` // Comma-separated list

	// Using variable expansion
	DataDir string `env:"DATA_DIR,expand" envDefault:"$HOME/data"`
}

func main() {
	// Simple usage - parse from environment
	cfg, err := env.Parse[Config](
		// Add a prefix to all variables (e.g., MYAPP_DEBUG instead of DEBUG)
		env.WithPrefix("MYAPP"),

		// Enable stage-based configuration (e.g., PROD_MYAPP_DEBUG takes precedence)
		// env.WithStage("PROD"),

		// Load .env files (doesn't override existing env vars)
		env.WithEnvFile(".env", ".env.local"),

		// Load JSON config files (lower precedence than env vars)
		env.WithJSONFile("config.json"),

		// Log when values are set
		env.WithOnSet(func(field, envVar string, value any, isDefault bool) {
			source := "env"
			if isDefault {
				source = "default"
			}
			fmt.Printf("  %s = %v (%s)\n", field, value, source)
		}),
	)

	if err != nil {
		// Handle aggregate errors
		if aggErr, ok := err.(*env.AggregateError); ok {
			log.Printf("Configuration errors (%d):", len(aggErr.Errors))
			for _, e := range aggErr.Errors {
				log.Printf("  - %v", e)
			}
			os.Exit(1)
		}
		log.Fatal(err)
	}

	fmt.Println("\nConfiguration loaded successfully:")
	fmt.Printf("  Server: %s:%d\n", cfg.Server.Host, cfg.Server.Port)
	fmt.Printf("  Database: %s@%s:%d/%s\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.Name)
	fmt.Printf("  Debug: %v\n", cfg.Debug)
	fmt.Printf("  DataDir: %s\n", cfg.DataDir)
}
