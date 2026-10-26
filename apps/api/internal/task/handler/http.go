package handler

import (
	"net/http"

	"teamflow/internal/httpx"

	"github.com/gin-gonic/gin"
)

// Handler adapts task HTTP requests onto the task use cases.
type Handler struct{ app TaskHandler }

func New(app TaskHandler) *Handler { return &Handler{app: app} }

func (h *Handler) Register(group *gin.RouterGroup) {
	group.POST("/tasks", h.create)
	group.GET("/tasks", h.list)
	group.GET("/tasks/:id", h.get)
	group.PUT("/tasks/:id", h.update)
	group.DELETE("/tasks/:id", h.remove)
}

func (h *Handler) create(c *gin.Context) {
	var body CreateTaskReq
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	body.ActorID = httpx.ActorID(c)
	resp, err := h.app.CreateTask(c.Request.Context(), &body)
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
	resp, err := h.app.ListTasks(c.Request.Context(), &ListTasksReq{ActorID: httpx.ActorID(c), ProjectID: c.Query("project_id"), Page: page, PageSize: size})
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
	resp, err := h.app.GetTask(c.Request.Context(), &GetTaskReq{ActorID: httpx.ActorID(c), TaskID: id})
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
	var body UpdateTaskReq
	if err := c.ShouldBindJSON(&body); err != nil {
		httpx.Fail(c, http.StatusBadRequest, "INVALID_ARGUMENT", "请求参数无效")
		return
	}
	body.ActorID, body.TaskID = httpx.ActorID(c), id
	resp, err := h.app.UpdateTask(c.Request.Context(), &body)
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
	resp, err := h.app.DeleteTask(c.Request.Context(), &DeleteTaskReq{ActorID: httpx.ActorID(c), TaskID: id})
	if err != nil {
		httpx.WriteError(c, err)
		return
	}
	httpx.Success(c, http.StatusOK, resp)
}
