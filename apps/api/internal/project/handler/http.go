package handler

import (
	"net/http"

	"teamflow/internal/httpx"

	"github.com/gin-gonic/gin"
)

// Handler adapts project HTTP requests onto the project use cases.
type Handler struct{ app ProjectHandler }

func New(app ProjectHandler) *Handler { return &Handler{app: app} }

func (h *Handler) Register(group *gin.RouterGroup) {
	group.POST("/teams/:id/projects", h.create)
	group.GET("/teams/:id/projects", h.list)
	group.GET("/projects/:id", h.get)
	group.PUT("/projects/:id", h.update)
	group.DELETE("/projects/:id", h.remove)
}

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
