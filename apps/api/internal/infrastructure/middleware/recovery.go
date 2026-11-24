package middleware

import (
	"fmt"
	"net/http"

	"teamflow/internal/infrastructure/httpx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Recovery 捕获请求处理中的 panic，记录错误，并在响应尚未写出时返回统一错误。
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if logger != nil {
					logger.Error("request panic recovered", zap.Any("request_id", c.GetString(httpx.RequestIDKey)), zap.String("panic_type", fmt.Sprintf("%T", recovered)))
				}
				if !c.Writer.Written() {
					httpx.Fail(c, http.StatusInternalServerError, "INTERNAL", "服务器内部错误")
				}
			}
		}()
		c.Next()
	}
}
