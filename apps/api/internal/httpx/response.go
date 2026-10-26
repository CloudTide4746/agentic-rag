package httpx

import (
	"errors"
	"fmt"
	"net/http"

	agentrepo "teamflow/internal/chat/repository"
	"teamflow/internal/faults"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

const (
	RequestIDKey = "request_id"
	LoggerKey    = "logger"
)

type errorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Success writes the platform success envelope.
func Success(c *gin.Context, status int, data any) {
	id, _ := c.Get(RequestIDKey)
	c.JSON(status, gin.H{"data": data, "request_id": id})
}

// Fail aborts the request chain with the platform error envelope.
func Fail(c *gin.Context, status int, code, message string) {
	id, _ := c.Get(RequestIDKey)
	c.AbortWithStatusJSON(status, gin.H{"error": errorBody{Code: code, Message: message}, "request_id": id})
}

// WriteError classifies a use-case error into the public status/code pair.
// Internal detail never reaches the response; it is logged with the request id.
func WriteError(c *gin.Context, err error) {
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
		logger, _ := c.Get(LoggerKey)
		if logger, ok := logger.(*zap.Logger); ok {
			id, _ := c.Get(RequestIDKey)
			logger.Error("internal API request failure", zap.Any("request_id", id), zap.String("code", code), zap.String("error_type", fmt.Sprintf("%T", err)))
		}
	}
	Fail(c, status, code, message)
}
