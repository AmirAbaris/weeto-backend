package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	"github.com/AmirAbaris/weeto-backend/internal/platform/email"
	applogger "github.com/AmirAbaris/weeto-backend/internal/platform/logger"
	"github.com/AmirAbaris/weeto-backend/internal/worker/notification"
	"github.com/joho/godotenv"
)

const (
	pollInterval = 5 * time.Second
	batchSize    = 20
)

func main() {
	_ = godotenv.Load()

	cfg, err := config.LoadConfig()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	logger := applogger.New(cfg.Env)
	slog.SetDefault(logger)

	sender, err := email.NewSender(cfg)
	if err != nil {
		slog.Error("email sender", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()

	pool, err := db.NewPool(ctx, cfg.DBURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := db.New(pool)
	processor := notification.NewProcessor(pool, queries, sender, cfg.FrontendURL)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	slog.Info("worker starting", "env", cfg.Env, "poll_interval", pollInterval)

	for {
		select {
		case <-stop:
			slog.Info("worker shutting down")
			return
		case <-ticker.C:
			processed, err := processor.ProcessBatch(ctx, batchSize)
			if err != nil {
				slog.Error("process batch", "err", err)
				continue
			}
			if processed > 0 {
				slog.Info("notifications processed", "count", processed)
			}
		}
	}
}
