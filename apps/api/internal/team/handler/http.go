package handler

import (
	"net/http"

	"teamflow/internal/infrastructure/httpx"

	"github.com/gin-gonic/gin"
)

// Handler adapts team HTTP requests onto the team use cases.
type Handler struct{ app TeamHandler }

// New 创建依赖团队用例的 HTTP 处理器。
func New(app TeamHandler) *Handler { return &Handler{app: app} }

// Register 注册团队的创建、列表、详情、更新和删除路由。
func (h *Handler) Register(group *gin.RouterGroup) {
	group.POST("/teams", h.create)
	group.GET("/teams", h.list)
	group.GET("/teams/:id", h.get)
	group.PUT("/teams/:id", h.update)
	group.DELETE("/teams/:id", h.remove)
}

// create 解析创建团队请求，注入当前用户身份并返回新建结果。
func (h *Handler) create(c *gin.Context) {
	var body CreateTeamReq
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	body.ActorID = httpx.ActorID(c)
	resp, err := h.app.CreateTeam(c.Request.Context(), &body)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// list 校验分页及相关查询参数，调用用例返回团队列表。
func (h *Handler) list(c *gin.Context) {
	page, size, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	resp, err := h.app.ListTeams(c.Request.Context(), &ListTeamsReq{ActorID: httpx.ActorID(c), Page: page, PageSize: size})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// get 校验路径 ID 并返回当前用户可访问的团队详情。
func (h *Handler) get(c *gin.Context) {
	id, ok := httpx.PathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.app.GetTeam(c.Request.Context(), &GetTeamReq{ActorID: httpx.ActorID(c), TeamID: id})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// update 校验路径 ID 和请求体，调用用例更新团队信息。
func (h *Handler) update(c *gin.Context) {
	id, ok := httpx.PathID(c, "id")
	if !ok {
		return
	}
	var body UpdateTeamReq
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	body.ActorID, body.TeamID = httpx.ActorID(c), id
	resp, err := h.app.UpdateTeam(c.Request.Context(), &body)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// remove 校验路径 ID，调用用例删除指定团队并返回结果。
func (h *Handler) remove(c *gin.Context) {
	id, ok := httpx.PathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.app.DeleteTeam(c.Request.Context(), &DeleteTeamReq{ActorID: httpx.ActorID(c), TeamID: id})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}
