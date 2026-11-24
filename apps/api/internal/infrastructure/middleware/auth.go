package middleware

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"teamflow/internal/auth"
	"teamflow/internal/infrastructure/faults"
	"teamflow/internal/infrastructure/httpx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Authenticator 提供令牌和当前登录会话的认证能力。
type Authenticator interface {
	Authenticate(ctx context.Context, token string) (*auth.Identity, error)
}

// Authenticate 校验 Bearer 凭据并注入已认证用户 ID，失败时终止请求链。
func Authenticate(authenticator Authenticator) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			httpx.Fail(c, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录")
			return
		}
		identity, err := authenticator.Authenticate(c.Request.Context(), parts[1])
		if err != nil {
			if errors.Is(err, faults.ErrUnauthenticated) {
				httpx.Fail(c, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录")
			} else {
				requestLogger, _ := c.Get(httpx.LoggerKey)
				if requestLogger, ok := requestLogger.(*zap.Logger); ok {
					requestLogger.Error("authentication backend unavailable", zap.Any("request_id", c.GetString(httpx.RequestIDKey)), zap.String("error_type", fmt.Sprintf("%T", err)))
				}
				httpx.Fail(c, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "认证服务暂不可用")
			}
			return
		}
		c.Set(httpx.ActorIDKey, identity.UserID)
		c.Next()
	}
}
