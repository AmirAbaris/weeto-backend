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
	smsplatform "github.com/AmirAbaris/weeto-backend/internal/platform/sms"
	applogger "github.com/AmirAbaris/weeto-backend/internal/platform/logger"
	notificationsvc "github.com/AmirAbaris/weeto-backend/internal/service/notification"
	"github.com/joho/godotenv"
)

const (
	pollInterval = 5 * time.Second
	batchSize    = 10
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

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DBURL)
	if err != nil {
		slog.Error("db", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	queries := db.New(pool)
	sender := smsplatform.NewSender(cfg)
	svc := notificationsvc.NewService(pool, queries, sender, cfg)

	runCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	slog.Info("worker starting", "env", cfg.Env, "sms_enabled", cfg.SMSAPIKey != "")

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-runCtx.Done():
			slog.Info("worker stopping")
			return
		case <-ticker.C:
			n, err := svc.ProcessPending(runCtx, batchSize)
			if err != nil {
				slog.Error("process pending", "err", err)
				continue
			}
			if n > 0 {
				slog.Info("processed notifications", "count", n)
			}
		}
	}
}
