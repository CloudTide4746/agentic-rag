package handler

import (
	"net/http"

	"teamflow/internal/httpx"

	"github.com/gin-gonic/gin"
)

// Handler adapts team HTTP requests onto the team use cases.
type Handler struct{ app TeamHandler }

func New(app TeamHandler) *Handler { return &Handler{app: app} }

func (h *Handler) Register(group *gin.RouterGroup) {
	group.POST("/teams", h.create)
	group.GET("/teams", h.list)
	group.GET("/teams/:id", h.get)
	group.PUT("/teams/:id", h.update)
	group.DELETE("/teams/:id", h.remove)
}

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
