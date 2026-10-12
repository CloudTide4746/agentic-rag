package handler

import (
	"context"
	"teamflow/internal/task"
	"time"
)

type CreateTaskReq struct {
	ActorID     uint       `json:"-"`
	ProjectID   string     `json:"project_id"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Priority    string     `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
}
type CreateTaskResp struct {
	Task task.Task `json:"task"`
}
type GetTaskReq struct {
	ActorID uint `json:"-"`
	TaskID  uint `json:"-"`
}
type GetTaskResp struct {
	Task task.Task `json:"task"`
}
type ListTasksReq struct {
	ActorID   uint   `json:"-"`
	ProjectID string `form:"project_id"`
	Page      int    `form:"page"`
	PageSize  int    `form:"page_size"`
}
type ListTasksResp struct {
	Items    []task.Task `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
type UpdateTaskReq struct {
	ActorID     uint       `json:"-"`
	TaskID      uint       `json:"-"`
	Version     *int       `json:"version"`
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Priority    *string    `json:"priority"`
	DueDate     *time.Time `json:"due_date"`
}
type UpdateTaskResp struct {
	Task task.Task `json:"task"`
}
type DeleteTaskReq struct {
	ActorID uint `json:"-"`
	TaskID  uint `json:"-"`
}
type DeleteTaskResp struct{}
type TaskHandler interface {
	CreateTask(ctx context.Context, req *CreateTaskReq) (*CreateTaskResp, error)
	GetTask(ctx context.Context, req *GetTaskReq) (*GetTaskResp, error)
	ListTasks(ctx context.Context, req *ListTasksReq) (*ListTasksResp, error)
	UpdateTask(ctx context.Context, req *UpdateTaskReq) (*UpdateTaskResp, error)
	DeleteTask(ctx context.Context, req *DeleteTaskReq) (*DeleteTaskResp, error)
}
