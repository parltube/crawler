// Package config loads runtime configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all crawler settings.
type Config struct {
	DatabaseURL       string
	WorkerConcurrency int
	BatchSize         int
}

// Load reads the configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL:       getenv("DATABASE_URL", ""),
		WorkerConcurrency: getenvInt("WORKER_CONCURRENCY", 10),
		BatchSize:         getenvInt("BATCH_SIZE", 50),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config: DATABASE_URL is required")
	}

	return cfg, nil
}

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}
