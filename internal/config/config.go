// Package config loads runtime configuration from the environment.
package config

import (
	"log/slog"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type LoggingConfig struct {
	LoggingLevel slog.Level `env:"LOGGING_LEVEL"`
}

// Config holds all crawler settings.
type Config struct {
	DatabaseURL       string `env:"DATABASE_URL",required,notEmpty`
	WorkerConcurrency int    `env:"WORKER_CONCURRENCY" envDefault:"10"`
	BatchSize         int    `env:"BATCH_SIZE" envDefault:"100"`
	ProxyCount        int    `env:"PROXY_COUNT" envDefault:"1"`
	ExportRaw         bool   `env:"EXPORT_RAW" envDefault:"false"`
	Debug             bool   `env:"DEBUG" envDefault:"false"`
	LoggingConfig     `envPrefix:"LOG_"`
}

// Load reads the configuration from environment variables.
func Load() (*Config, error) {
	err := godotenv.Load()
	if err != nil {
		return nil, err
	}

	cfg, err := env.ParseAs[Config]()

	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
