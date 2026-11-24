package handler

import (
	"net/http"

	"teamflow/internal/infrastructure/httpx"

	"github.com/gin-gonic/gin"
)

// Handler adapts project HTTP requests onto the project use cases.
type Handler struct{ app ProjectHandler }

// New 创建依赖项目用例的 HTTP 处理器。
func New(app ProjectHandler) *Handler { return &Handler{app: app} }

// Register 注册项目的创建、列表、详情、更新和删除路由。
func (h *Handler) Register(group *gin.RouterGroup) {
	group.POST("/teams/:id/projects", h.create)
	group.GET("/teams/:id/projects", h.list)
	group.GET("/projects/:id", h.get)
	group.PUT("/projects/:id", h.update)
	group.DELETE("/projects/:id", h.remove)
}

// create 解析创建项目请求，注入当前用户身份并返回新建结果。
func (h *Handler) create(c *gin.Context) {
	teamID, ok := httpx.PathID(c, "id")
	if !ok {
		return
	}
	var body CreateProjectReq
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	body.ActorID, body.TeamID = httpx.ActorID(c), teamID
	resp, err := h.app.CreateProject(c.Request.Context(), &body)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusCreated, resp)
}

// list 校验分页及相关查询参数，调用用例返回项目列表。
func (h *Handler) list(c *gin.Context) {
	teamID, ok := httpx.PathID(c, "id")
	if !ok {
		return
	}
	page, size, ok := httpx.ParsePage(c)
	if !ok {
		return
	}
	resp, err := h.app.ListProjects(c.Request.Context(), &ListProjectsReq{ActorID: httpx.ActorID(c), TeamID: teamID, Page: page, PageSize: size})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// get 校验路径 ID 并返回当前用户可访问的项目详情。
func (h *Handler) get(c *gin.Context) {
	id, ok := httpx.PathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.app.GetProject(c.Request.Context(), &GetProjectReq{ActorID: httpx.ActorID(c), ProjectID: id})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// update 校验路径 ID 和请求体，调用用例更新项目信息。
func (h *Handler) update(c *gin.Context) {
	id, ok := httpx.PathID(c, "id")
	if !ok {
		return
	}
	var body UpdateProjectReq
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	body.ActorID, body.ProjectID = httpx.ActorID(c), id
	resp, err := h.app.UpdateProject(c.Request.Context(), &body)
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}

// remove 校验路径 ID，调用用例删除指定项目并返回结果。
func (h *Handler) remove(c *gin.Context) {
	id, ok := httpx.PathID(c, "id")
	if !ok {
		return
	}
	resp, err := h.app.DeleteProject(c.Request.Context(), &DeleteProjectReq{ActorID: httpx.ActorID(c), ProjectID: id})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}
