package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AmirAbaris/weeto-backend/internal/config"
	"github.com/AmirAbaris/weeto-backend/internal/db"
	authhandler "github.com/AmirAbaris/weeto-backend/internal/handler/auth"
	docshandler "github.com/AmirAbaris/weeto-backend/internal/handler/docs"
	"github.com/AmirAbaris/weeto-backend/internal/handler/health"
	"github.com/AmirAbaris/weeto-backend/internal/middleware"
	applogger "github.com/AmirAbaris/weeto-backend/internal/platform/logger"
	authsvc "github.com/AmirAbaris/weeto-backend/internal/service/auth"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load() // ignore error in prod where .env doesnt exists

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

	healthHandler := health.NewHandler()
	docsHandler := docshandler.NewHandler()
	authService := authsvc.NewService(queries, cfg)
	authHandler := authhandler.NewHandler(authService)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /docs", docsHandler.UI)
	mux.HandleFunc("GET /openapi.yaml", docsHandler.Spec)
	mux.HandleFunc("GET /health", healthHandler.Live)
	mux.HandleFunc("POST /auth/register", authHandler.Register)
	mux.HandleFunc("POST /auth/login", authHandler.Login)
	mux.HandleFunc("POST /auth/refresh", authHandler.Refresh)
	mux.HandleFunc("POST /auth/logout", authHandler.Logout)

	handler := middleware.RequestID(
		middleware.Logging(logger, middleware.LoggingOptions{
			SkipPaths: map[string]struct{}{
				"/health": {},
			},
		})(mux),
	)

	server := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port, "env", cfg.Env)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}
