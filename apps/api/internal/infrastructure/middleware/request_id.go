package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strings"

	"teamflow/internal/infrastructure/httpx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// RequestID 校验或生成请求 ID，并将请求 ID 和可选日志器注入上下文。
func RequestID(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if !validRequestID(id) {
			id = requestID()
		}
		c.Set(httpx.RequestIDKey, id)
		if logger != nil {
			c.Set(httpx.LoggerKey, logger)
		}
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func requestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(bytes[:])
}

func validRequestID(id string) bool {
	if id == "" || len(id) > 128 {
		return false
	}
	for _, char := range id {
		if !(char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' || char >= '0' && char <= '9' || strings.ContainsRune("._-", char)) {
			return false
		}
	}
	return true
}
