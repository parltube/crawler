// Package app wires the crawler components together and runs them.
package app

import (
	"context"
	"log/slog"

	"github.com/crispyLL/parltube-crawler/internal/config"
)

// Run starts the crawler and blocks until ctx is cancelled.
func Run(ctx context.Context, cfg *config.Config, logger *slog.Logger) error {
	logger.Info("crawler started",
		"concurrency", cfg.WorkerConcurrency,
		"batch_size", cfg.BatchSize,
	)

	<-ctx.Done()

	logger.Info("crawler stopped")
	return nil
}
