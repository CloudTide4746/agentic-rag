package handler

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"teamflow/internal/faults"
	"teamflow/internal/httpx"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handler adapts HTTP requests for the auth module onto its use cases.
// It owns the module's public routes and the authentication middleware that
// protects every other module.
type Handler struct{ app AuthHandler }

func New(app AuthHandler) *Handler { return &Handler{app: app} }

func (h *Handler) Register(group *gin.RouterGroup) {
	group.POST("/auth/register", h.register)
	group.POST("/auth/login", h.login)
}

func (h *Handler) register(c *gin.Context) {
	var req RegisterReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	user, err := h.app.Register(c.Request.Context(), &req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, user)
}

func (h *Handler) login(c *gin.Context) {
	var req LoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	resp, err := h.app.Login(c.Request.Context(), &req)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// Middleware rejects invalid or unknown credentials before any use case runs
// and injects the verified actor id for downstream handlers.
func (h *Handler) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.SplitN(c.GetHeader("Authorization"), " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") || strings.TrimSpace(parts[1]) == "" {
			httpx.Fail(c, http.StatusUnauthorized, "UNAUTHENTICATED", "请先登录")
			return
		}
		identity, err := h.app.Authenticate(c.Request.Context(), parts[1])
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
