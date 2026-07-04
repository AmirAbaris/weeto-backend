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
	orghandler "github.com/AmirAbaris/weeto-backend/internal/handler/organization"
	availabilityhandler "github.com/AmirAbaris/weeto-backend/internal/handler/availability"
	interviewtypehandler "github.com/AmirAbaris/weeto-backend/internal/handler/interviewtype"
	publichandler "github.com/AmirAbaris/weeto-backend/internal/handler/public"
	"github.com/AmirAbaris/weeto-backend/internal/middleware"
	applogger "github.com/AmirAbaris/weeto-backend/internal/platform/logger"
	"github.com/AmirAbaris/weeto-backend/internal/server"
	authsvc "github.com/AmirAbaris/weeto-backend/internal/service/auth"
	availabilitysvc "github.com/AmirAbaris/weeto-backend/internal/service/availability"
	bookingsvc "github.com/AmirAbaris/weeto-backend/internal/service/booking"
	orgsvc "github.com/AmirAbaris/weeto-backend/internal/service/organization"
	interviewtypesvc "github.com/AmirAbaris/weeto-backend/internal/service/interviewtype"
	slotsvc "github.com/AmirAbaris/weeto-backend/internal/service/slot"
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
	orgService := orgsvc.NewService(queries, cfg)
	orgHandler := orghandler.NewHandler(orgService)
	slotService := slotsvc.NewService(queries)
	interviewTypeService := interviewtypesvc.NewService(queries, orgService, slotService)
	interviewTypeHandler := interviewtypehandler.NewHandler(interviewTypeService)
	availabilityService := availabilitysvc.NewService(pool, queries, orgService, slotService)
	availabilityHandler := availabilityhandler.NewHandler(availabilityService)
	bookingService := bookingsvc.NewService(pool, queries, orgService, slotService)
	publicHandler := publichandler.NewHandler(bookingService)

	mux := http.NewServeMux()
	server.Register(mux, cfg.JWTSecret, server.Handlers{
		Health:        healthHandler,
		Docs:          docsHandler,
		Auth:          authHandler,
		Organization:  orgHandler,
		InterviewType: interviewTypeHandler,
		Availability:  availabilityHandler,
		Public:        publicHandler,
	})

	handler := middleware.RequestID(
		middleware.Logging(logger, middleware.LoggingOptions{
			SkipPaths: map[string]struct{}{
				"/health": {},
			},
		})(mux),
	)

	httpServer := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}

	go func() {
		slog.Info("server starting", "port", cfg.Port, "env", cfg.Env)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("server failed", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown", "err", err)
	}
}
