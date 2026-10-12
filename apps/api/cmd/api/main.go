package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"teamflow/internal/platform"

	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic("initialize logger")
	}
	defer func() { _ = logger.Sync() }()
	address := os.Getenv("API_ADDR")
	if address == "" {
		logger.Fatal("API_ADDR must be set")
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	router, resources, err := platform.NewRouter(ctx, logger)
	if err != nil {
		logger.Fatal("initialize API", zap.Error(err))
	}
	defer func() {
		if err := resources.Redis.Close(); err != nil {
			logger.Warn("close Redis client", zap.Error(err))
		}
		if err := resources.SQLClose(); err != nil {
			logger.Warn("close SQL pool", zap.Error(err))
		}
	}()
	server := &http.Server{Addr: address, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	serveErr := make(chan error, 1)
	go func() {
		serveErr <- server.ListenAndServe()
	}()
	select {
	case err := <-serveErr:
		if err != nil && err != http.ErrServerClosed {
			logger.Error("server listen", zap.Error(err))
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("server shutdown", zap.Error(err))
		}
		if err := <-serveErr; err != nil && err != http.ErrServerClosed {
			logger.Error("server listen", zap.Error(err))
		}
	}
}
