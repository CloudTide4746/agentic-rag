package controller

import (
	"errors"
	"net/http"
	"teamflow/internal/dto"
	"teamflow/internal/model"
	"teamflow/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

type TaskController struct {
	svc service.TaskService
}

func NewTaskController(svc service.TaskService) *TaskController {
	return &TaskController{svc: svc}
}

type CreateTaskRequest struct {
	ProjectID   uint               `json:"project_id" binding:"required,gt=0"`
	Title       string             `json:"title" binding:"required,max=200"`
	Description string             `json:"description"`
	Priority    model.TaskPriority `json:"priority" binding:"omitempty,oneof=low medium high none"`
	DueDate     *time.Time         `json:"due_date"`
}

// @Summary CreateTask
// @Description POST /tasks
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body CreateTaskRequest true "Request body"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks [post]
func (c *TaskController) CreateTask(ctx *gin.Context) {
	var req CreateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task := &model.Task{
		ProjectID:   req.ProjectID,
		CreatorID:   ctx.GetUint("userID"),
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
	}
	if err := c.svc.CreateTask(task); err != nil {
		writeTaskError(ctx, err)
		return
	}
	ctx.JSON(http.StatusCreated, gin.H{"task": task})
}

// @Summary GetTaskDetail
// @Description GET /tasks/{id}
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id} [get]
func (c *TaskController) GetTaskDetail(ctx *gin.Context) {
	taskID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	task, err := c.svc.GetTask(taskID, ctx.GetUint("userID"))
	if err != nil {
		writeTaskError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

type UpdateTaskRequest struct {
	Title       *string             `json:"title" binding:"omitempty,max=200"`
	Description *string             `json:"description"`
	Priority    *model.TaskPriority `json:"priority" binding:"omitempty,oneof=low medium high none"`
	DueDate     *time.Time          `json:"due_date"`
}

// @Summary UpdateTask
// @Description PUT /tasks/{id}
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body UpdateTaskRequest true "Request body"
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id} [put]
func (c *TaskController) UpdateTask(ctx *gin.Context) {
	taskID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	var req UpdateTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := c.svc.UpdateTask(taskID, ctx.GetUint("userID"), service.UpdateTaskInput{
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		DueDate:     req.DueDate,
	})
	if err != nil {
		writeTaskError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

// @Summary DeleteTask
// @Description DELETE /tasks/{id}
// @Tags tasks
// @Accept json
// @Produce json
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id} [delete]
func (c *TaskController) DeleteTask(ctx *gin.Context) {
	taskID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	if err := c.svc.DeleteTask(taskID, ctx.GetUint("userID")); err != nil {
		writeTaskError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"message": "任务已删除"})
}

type ListTaskRequest struct {
	ProjectID uint `form:"project_id" binding:"required,gt=0"`
	Page      int  `form:"page,default=1" binding:"omitempty,min=1"`
	Size      int  `form:"size,default=10" binding:"omitempty,min=1,max=100"`
}

// @Summary GetTaskList
// @Description GET /tasks
// @Tags tasks
// @Accept json
// @Produce json
// @Param request query ListTaskRequest false "Query filters"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks [get]
func (c *TaskController) GetTaskList(ctx *gin.Context) {
	var req ListTaskRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	tasks, total, err := c.svc.GetTaskList(
		req.ProjectID,
		ctx.GetUint("userID"),
		req.Page,
		req.Size,
	)
	if err != nil {
		writeTaskError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{
		"tasks": tasks,
		"total": total,
		"page":  normalizedPage(req.Page),
		"size":  normalizedSize(req.Size),
	})
}

type ChangeTaskStatusRequest struct {
	Status model.TaskStatus `json:"status" binding:"required,oneof=todo in_progress review done"`
}

// @Summary ChangeTaskStatus
// @Description PUT /tasks/{id}/status
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body ChangeTaskStatusRequest true "Request body"
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id}/status [put]
func (c *TaskController) ChangeTaskStatus(ctx *gin.Context) {
	taskID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	var req ChangeTaskStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := c.svc.ChangeTaskStatus(taskID, ctx.GetUint("userID"), req.Status)
	if err != nil {
		writeTaskError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

// @Summary AssignTask
// @Description PATCH /tasks/{id}/assignee
// @Tags tasks
// @Accept json
// @Produce json
// @Param request body dto.AssignTaskRequest true "Request body"
// @Param id path string true "id ID"
// @Success 200 {object} APIResponse
// @Failure 400 {object} APIResponse
// @Security BearerAuth
// @Router /tasks/{id}/assignee [patch]
func (c *TaskController) AssignTask(ctx *gin.Context) {
	taskID, ok := parseID(ctx, "id")
	if !ok {
		return
	}
	id := ctx.GetUint("userID")
	var req dto.AssignTaskRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	task, err := c.svc.AssignTask(taskID, req.AssigneeID, ctx, id)
	if err != nil {
		writeTaskError(ctx, err)
		return
	}
	ctx.JSON(http.StatusOK, gin.H{"task": task})
}

func writeTaskError(ctx *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidTask):
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrTaskForbidden):
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrTaskNotFound):
		ctx.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrTaskConflict):
		ctx.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	case errors.Is(err, service.ErrInvalidTransition),
		errors.Is(err, service.ErrAssigneeNotInProject):
		ctx.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
	default:
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

func normalizedPage(page int) int {
	if page < 1 {
		return 1
	}
	return page
}

func normalizedSize(size int) int {
	if size < 1 {
		return 10
	}
	if size > 100 {
		return 100
	}
	return size
}
