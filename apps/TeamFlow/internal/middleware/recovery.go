package middleware

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery 中间件，用于捕获并记录 panic 异常
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("recovery panic", zap.Any("err", r), zap.String("path", c.Request.URL.Path), zap.String("method", c.Request.Method))
			}

		}()
		c.Next()
	}
}
