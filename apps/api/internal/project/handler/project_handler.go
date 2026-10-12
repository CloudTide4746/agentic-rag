package handler

import (
	"context"
	"teamflow/internal/project"
	"time"
)

type CreateProjectReq struct {
	ActorID     uint       `json:"-"`
	TeamID      uint       `json:"-"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
}
type CreateProjectResp struct {
	Project project.Project `json:"project"`
}
type ListProjectsReq struct {
	ActorID  uint `json:"-"`
	TeamID   uint `json:"-"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
}
type ListProjectsResp struct {
	Items    []project.Project `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
}
type GetProjectReq struct {
	ActorID   uint `json:"-"`
	ProjectID uint `json:"-"`
}
type GetProjectResp struct {
	Project project.Project `json:"project"`
}
type UpdateProjectReq struct {
	ActorID     uint       `json:"-"`
	ProjectID   uint       `json:"-"`
	Name        *string    `json:"name"`
	Description *string    `json:"description"`
	Status      *string    `json:"status"`
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`
}
type UpdateProjectResp struct {
	Project project.Project `json:"project"`
}
type DeleteProjectReq struct {
	ActorID   uint `json:"-"`
	ProjectID uint `json:"-"`
}
type DeleteProjectResp struct{}

type ProjectHandler interface {
	CreateProject(context.Context, *CreateProjectReq) (*CreateProjectResp, error)
	ListProjects(context.Context, *ListProjectsReq) (*ListProjectsResp, error)
	GetProject(context.Context, *GetProjectReq) (*GetProjectResp, error)
	UpdateProject(context.Context, *UpdateProjectReq) (*UpdateProjectResp, error)
	DeleteProject(context.Context, *DeleteProjectReq) (*DeleteProjectResp, error)
}
