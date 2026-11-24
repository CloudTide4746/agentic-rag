package handler

import (
	"net/http"

	"teamflow/internal/infrastructure/httpx"

	"github.com/gin-gonic/gin"
)

// Handler adapts task HTTP requests onto the task use cases.
type Handler struct{ app TaskHandler }

// New 创建依赖任务用例的 HTTP 处理器。
func New(app TaskHandler) *Handler { return &Handler{app: app} }

// Register 注册任务的创建、列表、详情、更新和删除路由。
func (h *Handler) Register(group *gin.RouterGroup) {
	group.POST("/tasks", h.create)
	group.GET("/tasks", h.list)
	group.GET("/tasks/:id", h.get)
	group.PUT("/tasks/:id", h.update)
	group.DELETE("/tasks/:id", h.remove)
}

// create 解析创建任务请求，注入当前用户身份并返回新建结果。
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

// list 校验分页及相关查询参数，调用用例返回任务列表。
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

// get 校验路径 ID 并返回当前用户可访问的任务详情。
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

// update 校验路径 ID 和请求体，调用用例更新任务信息。
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

// remove 校验路径 ID，调用用例删除指定任务并返回结果。
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
