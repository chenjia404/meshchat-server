package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"meshchat-server/internal/app"
	"meshchat-server/internal/config"
	"meshchat-server/pkg/logx"
)

func main() {
	cfg := config.Load()
	logger := logx.New(cfg.LogLevel)

	application, err := app.New(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize application", slog.Any("error", err))
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := application.Run(ctx); err != nil {
		logger.Error("application stopped with error", slog.Any("error", err))
		os.Exit(1)
	}
}
