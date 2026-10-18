// Command crawler is the entrypoint of the YouTube crawler.
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/crispyLL/parltube-crawler/internal/config"
	"github.com/crispyLL/parltube-crawler/internal/crawler"
	"github.com/crispyLL/parltube-crawler/internal/proxy"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	proxy, err := proxy.NewRevolvingProxy(cfg)

	if err != nil {
		logger.Error("Error loading proxies", "err", err)
		os.Exit(1)
	}

	crawler, err := crawler.NewCrawler(1, proxy)

	if err := crawler.Run(ctx, cfg); err != nil {
		logger.Error("run", "err", err)
		os.Exit(1)
	}
}
