package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Bahjat/page-insight-tool/backend/internal/analyzer"
	"github.com/Bahjat/page-insight-tool/backend/internal/pageinsight"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/config"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/logger"
	"github.com/Bahjat/page-insight-tool/backend/internal/platform/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %v\n", err)
		os.Exit(1)
	}

	log := logger.New(cfg.LogLevel)

	fetcher := pageinsight.NewHTTPClient()
	checker := pageinsight.NewLinkChecker(cfg.LinkCheckConcurrency)
	engine := pageinsight.NewEngine(fetcher, checker)
	svc := analyzer.NewService(engine, log)
	transport := analyzer.NewTransport(svc, log)

	mux := http.NewServeMux()
	transport.RegisterRoutes(mux)
	handler := middleware.CORS(mux)
	handler = middleware.Logging(log)(handler)
	handler = middleware.RequestID(handler)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      handler,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	log.Info("server starting", "port", cfg.Port)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	errCh := make(chan error, 1)
	go func() {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		log.Error("server error", "error", err)
		os.Exit(1)
	case sig := <-quit:
		log.Info("shutting down gracefully", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)

	if err := srv.Shutdown(ctx); err != nil {
		cancel()
		log.Error("forced shutdown", "error", err)
		os.Exit(1)
	}
	defer cancel()
	log.Info("server stopped")
}
