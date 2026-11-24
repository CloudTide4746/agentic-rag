package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// BodyLimit 限制请求体的最大读取字节数，读取错误由下游处理器映射为响应。
func BodyLimit(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}
