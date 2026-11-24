package platform

import (
	"teamflow/internal/infrastructure/middleware"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// requestIDMiddleware 保留包内测试入口，生产组装直接使用 middleware.RequestID。
func requestIDMiddleware(loggers ...*zap.Logger) gin.HandlerFunc {
	var logger *zap.Logger
	if len(loggers) > 0 {
		logger = loggers[0]
	}
	return middleware.RequestID(logger)
}

// recoveryMiddleware 保留包内测试入口，恢复实现统一委托给 middleware 包。
func recoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return middleware.Recovery(logger)
}
