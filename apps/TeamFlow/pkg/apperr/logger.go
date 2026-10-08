package apperr

import (
	"errors"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// LogError 记录错误日志，自动区分业务错误和系统错误
// 业务错误用 Warn 级别（预期内），系统错误用 Error 级别（需关注）
func LogError(logger *zap.Logger, c *gin.Context, err error) {
	fields := []zap.Field{
		zap.String("path", c.Request.URL.Path),
		zap.String("method", c.Request.Method),
		zap.String("client_ip", c.ClientIP()),
		zap.Error(err),
	}

	var appErr *AppError
	if errors.As(err, &appErr) {
		// 业务错误：记录业务码和说明，不记录堆栈（噪音）
		fields = append(fields,
			zap.Int("biz_code", appErr.Code),
			zap.String("biz_message", appErr.Message),
		)
		logger.Warn("business error", fields...)
	} else {
		// 系统错误：记录完整堆栈，便于排查
		fields = append(fields,
			zap.String("stack", string(debug.Stack())),
		)
		logger.Error("system error", fields...)
	}
}
