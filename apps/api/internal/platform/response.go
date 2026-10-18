package platform

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"teamflow/internal/faults"

	agentrepo "teamflow/internal/chat/repository"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const requestIDKey = "request_id"

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
			c.Set("logger", loggers[0])
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
					fail(c, http.StatusInternalServerError, "INTERNAL", "服务器内部错误")
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

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func success(c *gin.Context, status int, data any) {
	id, _ := c.Get(requestIDKey)
	c.JSON(status, gin.H{"data": data, "request_id": id})
}

func fail(c *gin.Context, status int, code, message string) {
	id, _ := c.Get(requestIDKey)
	c.AbortWithStatusJSON(status, gin.H{"error": errorBody{Code: code, Message: message}, "request_id": id})
}

func writeError(c *gin.Context, err error) {
	if err == nil {
		return
	}
	status, code, message := http.StatusInternalServerError, "INTERNAL", "服务器内部错误"
	switch {
	case errors.Is(err, faults.ErrValidation):
		status, code, message = http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效"
	case errors.Is(err, faults.ErrNotFound):
		status, code, message = http.StatusNotFound, "NOT_FOUND", "资源不存在"
	case errors.Is(err, faults.ErrForbidden):
		status, code, message = http.StatusForbidden, "FORBIDDEN", "无访问权限"
	case errors.Is(err, faults.ErrConflict):
		status, code, message = http.StatusConflict, "CONFLICT", "资源冲突"
	case errors.Is(err, faults.ErrUnauthenticated):
		status, code, message = http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录"
	// Agent runtime failures become stable public codes so the browser never
	// sees internal Agent detail (ADR-0003 "错误映射").
	case errors.Is(err, agentrepo.ErrAgentRejected):
		status, code, message = http.StatusBadGateway, "AGENT_REJECTED", "问答服务拒绝了本次请求"
	case errors.Is(err, agentrepo.ErrAgentTimeout):
		status, code, message = http.StatusGatewayTimeout, "AGENT_TIMEOUT", "问答服务响应超时"
	case errors.Is(err, agentrepo.ErrAgentUnavailable), errors.Is(err, agentrepo.ErrAgentUnreachable):
		status, code, message = http.StatusServiceUnavailable, "AGENT_UNAVAILABLE", "问答服务暂不可用"
	}
	if status >= http.StatusInternalServerError {
		logger, _ := c.Get("logger")
		if logger, ok := logger.(*zap.Logger); ok {
			id, _ := c.Get(requestIDKey)
			logger.Error("internal API request failure", zap.Any("request_id", id), zap.String("code", code), zap.String("error_type", fmt.Sprintf("%T", err)))
		}
	}
	fail(c, status, code, message)
}
