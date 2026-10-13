package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"teamflow/internal/faults"
	"teamflow/internal/task"
	"teamflow/internal/task/handler"	
)

type Repository interface {
	Create(context.Context, task.Task) (*task.Task, error)
	ListByProject(context.Context, uint, int, int) ([]task.Task, int64, error)
	Get(context.Context, uint) (*task.Task, error)
	Update(context.Context, uint, int, map[string]any) (*task.Task, error)
	Delete(context.Context, uint) error
}
type ProjectAccess interface {
	IsMember(context.Context, uint, uint) (bool, error)
}
type TaskService struct {
	repo     Repository
	projects ProjectAccess
}

var _ handler.TaskHandler = (*TaskService)(nil)

func NewTaskService(repo Repository, projects ProjectAccess) *TaskService {
	return &TaskService{repo: repo, projects: projects}
}
// CreateTask 创建任务
func (s *TaskService) CreateTask(ctx context.Context, req *handler.CreateTaskReq) (*handler.CreateTaskResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	projectID, err := strconv.ParseUint(req.ProjectID, 10, 64)
	if err != nil || projectID == 0 || strconv.FormatUint(projectID, 10) != req.ProjectID {
		return nil, fmt.Errorf("project_id is invalid: %w", faults.ErrValidation)
	}
	title := strings.TrimSpace(req.Title)
	if title == "" || utf8.RuneCountInString(title) > 200 || utf8.RuneCountInString(req.Description) > 10000 {
		return nil, faults.ErrValidation
	}
	priority := req.Priority
	if priority == "" {
		priority = "medium"
	}
	if priority != "low" && priority != "medium" && priority != "high" && priority != "none" {
		return nil, faults.ErrValidation
	}
	member, err := s.projects.IsMember(ctx, uint(projectID), req.ActorID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, faults.ErrForbidden
	}
	created, err := s.repo.Create(ctx, task.Task{Title: title, Description: req.Description, ProjectID: fmt.Sprint(projectID), CreatorID: fmt.Sprint(req.ActorID), Priority: priority, Status: "todo", DueDate: req.DueDate})
	if err != nil {
		return nil, err
	}
	return &handler.CreateTaskResp{Task: *created}, nil
}
// GetTask 获取任务
func (s *TaskService) GetTask(ctx context.Context, req *handler.GetTaskReq) (*handler.GetTaskResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	item, err := s.requireMember(ctx, req.TaskID, req.ActorID)
	if err != nil {
		return nil, err
	}
	return &handler.GetTaskResp{Task: *item}, nil
}
// ListTasks 获取任务列表
func (s *TaskService) ListTasks(ctx context.Context, req *handler.ListTasksReq) (*handler.ListTasksResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	projectID, err := strconv.ParseUint(req.ProjectID, 10, 64)
	if err != nil || projectID == 0 || strconv.FormatUint(projectID, 10) != req.ProjectID {
		return nil, fmt.Errorf("project_id is required: %w", faults.ErrValidation)
	}
	member, err := s.projects.IsMember(ctx, uint(projectID), req.ActorID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, faults.ErrForbidden
	}
	page, size := req.Page, req.PageSize
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	items, total, err := s.repo.ListByProject(ctx, uint(projectID), page, size)
	if err != nil {
		return nil, err
	}
	return &handler.ListTasksResp{Items: items, Total: total, Page: page, PageSize: size}, nil
}
// UpdateTask 更新任务
func (s *TaskService) UpdateTask(ctx context.Context, req *handler.UpdateTaskReq) (*handler.UpdateTaskResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	item, err := s.requireMember(ctx, req.TaskID, req.ActorID)
	if err != nil {
		return nil, err
	}
	if req.Version == nil || *req.Version < 0 {
		return nil, fmt.Errorf("version is required: %w", faults.ErrValidation)
	}
	if item.Version != *req.Version {
		return nil, faults.ErrConflict
	}
	fields := map[string]any{}
	if req.Title != nil {
		title := strings.TrimSpace(*req.Title)
		if title == "" || utf8.RuneCountInString(title) > 200 {
			return nil, faults.ErrValidation
		}
		fields["title"] = title
	}
	if req.Description != nil {
		if utf8.RuneCountInString(*req.Description) > 10000 {
			return nil, faults.ErrValidation
		}
		fields["description"] = *req.Description
	}
	if req.Priority != nil {
		if *req.Priority != "low" && *req.Priority != "medium" && *req.Priority != "high" && *req.Priority != "none" {
			return nil, faults.ErrValidation
		}
		fields["priority"] = *req.Priority
	}
	if req.DueDate != nil {
		fields["due_date"] = req.DueDate
	}
	if len(fields) == 0 {
		return nil, faults.ErrValidation
	}
	updated, err := s.repo.Update(ctx, req.TaskID, *req.Version, fields)
	if err != nil {
		return nil, err
	}
	return &handler.UpdateTaskResp{Task: *updated}, nil
}
// DeleteTask 删除任务
func (s *TaskService) DeleteTask(ctx context.Context, req *handler.DeleteTaskReq) (*handler.DeleteTaskResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	if _, err := s.requireMember(ctx, req.TaskID, req.ActorID); err != nil {
		return nil, err
	}
	if err := s.repo.Delete(ctx, req.TaskID); err != nil {
		return nil, err
	}
	return &handler.DeleteTaskResp{}, nil
}
// requireMember 检查用户是否是任务的成员
func (s *TaskService) requireMember(ctx context.Context, taskID, userID uint) (*task.Task, error) {
	item, err := s.repo.Get(ctx, taskID)
	if err != nil {
		return nil, err
	}
	projectID, _ := strconv.ParseUint(item.ProjectID, 10, 64)
	member, err := s.projects.IsMember(ctx, uint(projectID), userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, faults.ErrForbidden
	}
	return item, nil
}
