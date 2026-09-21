package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agentic-rag/apps/api/internal/config"
	"agentic-rag/apps/api/internal/platform"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	log, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	defer func() { _ = log.Sync() }()
	gin.SetMode(gin.ReleaseMode)
	cfg := config.Load()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	server := &http.Server{Addr: cfg.HTTPAddr, Handler: platform.NewRouter(log), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	result := make(chan error, 1)
	go func() { result <- server.ListenAndServe() }()
	log.Info("api_starting", zap.String("addr", cfg.HTTPAddr))
	select {
	case err := <-result:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Fatal("api_failed", zap.Error(err))
		}
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Error("shutdown_failed", zap.Error(err))
			_ = server.Close()
		}
	}
}
