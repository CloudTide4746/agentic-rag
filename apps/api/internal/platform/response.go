package platform

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"

	"teamflow/internal/httpx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const requestIDKey = httpx.RequestIDKey

func requestID() string {
	var bytes [16]byte
	if _, err := rand.Read(bytes[:]); err != nil {
		return "unavailable"
	}
	return hex.EncodeToString(bytes[:])
}

func requestIDMiddleware(loggers ...*zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if !validRequestID(id) {
			id = requestID()
		}
		c.Set(requestIDKey, id)
		if len(loggers) > 0 && loggers[0] != nil {
			c.Set(httpx.LoggerKey, loggers[0])
		}
		c.Header("X-Request-ID", id)
		c.Next()
	}
}

func jsonBodyLimitMiddleware(limit int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, limit)
		}
		c.Next()
	}
}

func recoveryMiddleware(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				if logger != nil {
					logger.Error("request panic recovered", zap.Any("request_id", c.GetString(requestIDKey)), zap.String("panic_type", fmt.Sprintf("%T", recovered)))
				}
				if !c.Writer.Written() {
					httpx.Fail(c, http.StatusInternalServerError, "INTERNAL", "服务器内部错误")
				}
			}
		}()
		c.Next()
	}
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
