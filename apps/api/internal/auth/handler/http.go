package handler

import (
	"net/http"

	"teamflow/internal/infrastructure/httpx"
	"teamflow/internal/infrastructure/middleware"

	"github.com/gin-gonic/gin"
)

// Handler adapts HTTP requests for the auth module onto its use cases.
// It owns the module's public registration and login routes.
type Handler struct{ app AuthHandler }

// New 创建依赖认证用例的 HTTP 处理器。
func New(app AuthHandler) *Handler { return &Handler{app: app} }

// Register 注册用户注册和登录的公开路由。
func (h *Handler) Register(group *gin.RouterGroup) {
	group.POST("/auth/register", h.register)
	group.POST("/auth/login", h.login)
}

// register 解析注册请求，调用注册用例并返回新用户信息。
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

// login 解析登录请求，调用登录用例并返回令牌及用户信息。
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

// Middleware 保留现有调用入口，认证实现统一委托给 middleware 包。
// Deprecated: 组装层应直接使用 middleware.Authenticate。
func (h *Handler) Middleware() gin.HandlerFunc {
	return middleware.Authenticate(h.app)
}
