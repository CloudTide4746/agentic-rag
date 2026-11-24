package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"teamflow/internal/infrastructure/config"
	"teamflow/internal/infrastructure/platform"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// main 加载配置并组装 API 服务，监听退出信号以优雅关闭 HTTP 服务和基础设施连接。
func main() {
	// 解析命令行参数
	// 从命令行参数中获取配置文件路径
	configPath := flag.String("config", "", "YAML config path (default: config.yaml if present)")
	flag.Parse()

	// 初始化日志记录器
	logger, err := zap.NewProduction()
	if err != nil {
		panic("initialize logger")
	}

	// 延迟关闭日志记录器
	defer func() { _ = logger.Sync() }()
	// 加载配置文件
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Fatal("load configuration", zap.Error(err))
	}
	gin.SetMode(cfg.Server.GinMode)

	// 初始化路由
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// 延迟关闭信号通知上下文
	defer stop()
	router, resources, err := platform.NewRouter(ctx, logger, cfg)
	if err != nil {
		logger.Fatal("initialize API", zap.Error(err))
	}
	// 延迟关闭资源
	defer func() {
		if err := resources.Redis.Close(); err != nil {
			logger.Warn("close Redis client", zap.Error(err))
		}
		if err := resources.SQLClose(); err != nil {
			logger.Warn("close SQL pool", zap.Error(err))
		}
	}()

	// 启动 HTTP 服务器
	server := &http.Server{Addr: cfg.Server.Addr, Handler: router, ReadHeaderTimeout: 5 * time.Second}
	logger.Info("starting API", zap.String("address", cfg.Server.Addr), zap.String("gin_mode", cfg.Server.GinMode))
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
