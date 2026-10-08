package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"
	"sync"
	"teamflow/internal/cache"
	"teamflow/internal/event"
	"teamflow/internal/model"
	"teamflow/internal/publisher"

	"teamflow/internal/repository"
	"time"

	"gorm.io/gorm"
)

var (
	ErrTaskNotFound         = errors.New("task not found")
	ErrTaskForbidden        = errors.New("task operation forbidden")
	ErrTaskConflict         = errors.New("task was modified concurrently")
	ErrInvalidTask          = errors.New("invalid task")
	ErrInvalidTransition    = errors.New("invalid status transition")
	ErrAssigneeNotInProject = errors.New("assignee is not a project member")
)

type UpdateTaskInput struct {
	Title       *string
	Description *string
	Priority    *model.TaskPriority
	DueDate     *time.Time
}

type TaskService interface {
	CreateTask(task *model.Task) error
	UpdateTask(id, operatorID uint, input UpdateTaskInput) (*model.Task, error)
	DeleteTask(id, operatorID uint) error
	GetTask(id, operatorID uint) (*model.Task, error)
	GetTaskList(projectID, operatorID uint, page, size int) ([]*model.Task, int64, error)
	ChangeTaskStatus(id, operatorID uint, newStatus model.TaskStatus) (*model.Task, error)
	AssignTask(id uint, assigneeID uint, ctx context.Context, operatorID uint) (*model.Task, error)
}

type taskService struct {
	tx        *gorm.DB
	repo      repository.TaskRepository
	notifier  NotificationService
	publisher publisher.Publisher
	mu        sync.Mutex
}

type taskListCacheValue struct {
	Tasks []*model.Task `json:"tasks"`
	Total int64         `json:"total"`
}

func NewTaskService(tx *gorm.DB, repo repository.TaskRepository, notifier NotificationService, publisher publisher.Publisher) TaskService {
	return &taskService{tx: tx, repo: repo, notifier: notifier, publisher: publisher}
}

func (s *taskService) CreateTask(task *model.Task) error {
	if task == nil {
		return fmt.Errorf("%w: task is required", ErrInvalidTask)
	}
	task.Title = strings.TrimSpace(task.Title)
	if task.Title == "" {
		return fmt.Errorf("%w: title is required", ErrInvalidTask)
	}
	if task.ProjectID == 0 {
		return fmt.Errorf("%w: project_id is required", ErrInvalidTask)
	}
	if task.CreatorID == 0 {
		return fmt.Errorf("%w: creator_id is required", ErrInvalidTask)
	}
	if task.Priority == "" {
		task.Priority = model.TaskPriorityMedium
	}
	if !model.IsValidTaskPriority(task.Priority) {
		return fmt.Errorf("%w: unsupported priority", ErrInvalidTask)
	}
	if task.Status == "" {
		task.Status = model.TaskStatusTodo
	}
	if !model.IsValidTaskStatus(task.Status) {
		return fmt.Errorf("%w: unsupported status", ErrInvalidTask)
	}
	if _, err := s.requireProjectMember(task.ProjectID, task.CreatorID); err != nil {
		return err
	}
	if err := s.repo.Create(task); err != nil {
		return fmt.Errorf("create task: %w", err)
	}
	_ = cache.InvalidateTask(task.ID, task.ProjectID)
	return nil
}

func (s *taskService) GetTask(id, operatorID uint) (*model.Task, error) {
	task, err := s.getTask(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireProjectMember(task.ProjectID, operatorID); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *taskService) GetTaskList(projectID, operatorID uint, page, size int) ([]*model.Task, int64, error) {

	if projectID == 0 {
		return nil, 0, fmt.Errorf("%w: project_id is required", ErrInvalidTask)
	}
	if _, err := s.requireProjectMember(projectID, operatorID); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 10
	}
	if size > 100 {
		size = 100
	}

	// 防止缓存击穿
	// 	1.请求查redis，没查到，如果查到，命中
	cacheKey := cache.TaskListKey(projectID, page, size)
	var cached taskListCacheValue
	if ok, _ := cache.GetResourceCache(cacheKey, &cached); ok {
		return cached.Tasks, cached.Total, nil
	}

	// 2.抢锁
	s.mu.Lock()
	defer s.mu.Unlock()

	// 3.抢到锁之后，再查一遍redis，如果有直接命中

	var cached2 taskListCacheValue
	if ok, _ := cache.GetResourceCache(cacheKey, &cached2); ok {
		return cached2.Tasks, cached2.Total, nil
	}

	// 4.查数据库
	tasks, total, err := s.repo.List(projectID, page, size)
	if err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}

	// 5.写redis

	_ = cache.SetListCache(cacheKey, taskListCacheValue{Tasks: tasks, Total: total})
	return tasks, total, nil
}

func (s *taskService) UpdateTask(id, operatorID uint, input UpdateTaskInput) (*model.Task, error) {
	task, err := s.getTask(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireProjectMember(task.ProjectID, operatorID); err != nil {
		return nil, err
	}

	updates := make(map[string]interface{}, 4)
	if input.Title != nil {
		title := strings.TrimSpace(*input.Title)
		if title == "" {
			return nil, fmt.Errorf("%w: title cannot be empty", ErrInvalidTask)
		}
		updates["title"] = title
	}
	if input.Description != nil {
		updates["description"] = *input.Description
	}
	if input.Priority != nil {
		if !model.IsValidTaskPriority(*input.Priority) {
			return nil, fmt.Errorf("%w: unsupported priority", ErrInvalidTask)
		}
		updates["priority"] = *input.Priority
	}
	if input.DueDate != nil {
		updates["due_date"] = *input.DueDate
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("%w: at least one field is required", ErrInvalidTask)
	}
	if err := s.updateFields(task, updates); err != nil {
		return nil, err
	}
	return s.getTask(id)
}

func (s *taskService) DeleteTask(id, operatorID uint) error {
	task, err := s.getTask(id)
	if err != nil {
		return err
	}
	member, err := s.requireProjectMember(task.ProjectID, operatorID)
	if err != nil {
		return err
	}
	if task.CreatorID != operatorID && !model.HasAdminPermission(member.Role) {
		return ErrTaskForbidden
	}
	if err := s.repo.Delete(id); err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	_ = cache.InvalidateTask(task.ID, task.ProjectID)
	_ = cache.InvalidateTaskComments(task.ID)
	return nil
}

func (s *taskService) ChangeTaskStatus(id, operatorID uint, newStatus model.TaskStatus) (*model.Task, error) {
	task, err := s.getTask(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireProjectMember(task.ProjectID, operatorID); err != nil {
		return nil, err
	}
	if !model.IsValidTaskStatus(newStatus) || !model.CanTransition(task.Status, newStatus) {
		return nil, ErrInvalidTransition
	}
	oldStatus := task.Status
	if err := s.updateFields(task, map[string]interface{}{"status": newStatus}); err != nil {
		return nil, err
	}
	updated, err := s.getTask(id)
	if err != nil {
		return nil, err
	}
	if err := s.notifier.OnTaskStatusChange(updated, oldStatus, newStatus); err != nil {
		log.Printf("notify task status change failed: %v", err)
	}
	return updated, nil
}

// AssignTaskv2 分配任务v2```go
// 1. 完成现有业务校验
// 2. updateFields(task, {"assignee_id": assigneeID})
func (s *taskService) AssignTask(id uint, assigneeID uint, ctx context.Context, operatorID uint) (*model.Task, error) {
	// 1. 校验任务是否存在
	task, err := s.getTask(id)
	if err != nil {
		return nil, err
	}
	if _, err := s.requireProjectMember(task.ProjectID, operatorID); err != nil {
		return nil, err
	}
	//2. 校验assignee是否存在
	if _, err := s.requireProjectMember(task.ProjectID, assigneeID); err != nil {
		return nil, err
	}
	//身份校验
	operator, err := s.requireProjectMember(task.ProjectID, operatorID)
	if err != nil {
		return nil, err
	}
	if !model.HasAdminPermission(operator.Role) {
		return nil, ErrTaskForbidden
	}

	// updated, err := s.getTask(id)
	// if err != nil {
	// 	return nil, err
	// }
	// 4. 数据库成功后才发布事件。Consumer 按 EventType 分发，因此消息必须保留完整封包。
	message := event.NewTaskAssigned(task.ID, task.ProjectID, assigneeID, operatorID)
	body, err := json.Marshal(message)
	if err != nil {
		// Level 2 的策略：任务已指派，记录错误但不应让这次分配失败。
		log.Printf("marshal task assigned event failed: %v", err)
		return task, nil
	}
	now := time.Now().UTC()
	err = s.tx.Transaction(func(tx *gorm.DB) error {
		updated, err := s.repo.UpdateFieldsTx(
			tx,
			task.ID,
			task.Version,
			map[string]interface{}{"assignee_id": assigneeID, "assigned_at": now},
		)
		if err != nil {
			return err
		}
		if !updated {
			return ErrTaskConflict
		}
		return tx.Create(&model.OutboxEvent{
			EventID:       message.EventID,
			EventType:     message.EventType,
			Exchange:      event.EventsExchange,
			RoutingKey:    event.TaskAssignedRouting,
			Payload:       body,
			Status:        model.OutboxPending,
			Attempts:      0,
			NextAttemptAt: now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	if err := cache.InvalidateTask(task.ID, task.ProjectID); err != nil {
		log.Printf("invalidate task cache failed: %v", err)
	}

	updatedTask, err := s.repo.GetByID(task.ID)
	if err != nil {
		return nil, fmt.Errorf("query assigned task: %w", err)
	}
	return updatedTask, nil
}

func (s *taskService) getTask(id uint) (*model.Task, error) {
	if id == 0 {
		return nil, fmt.Errorf("%w: id is required", ErrInvalidTask)
	}
	cacheKey := cache.TaskKey(id)
	var cached model.Task

	if ok, _ := cache.GetResourceCache(cacheKey, &cached); ok {
		return &cached, nil
	}
	task, err := s.repo.GetByID(id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTaskNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query task: %w", err)
	}
	_ = cache.SetDetailCache(cacheKey, task)
	return task, nil
}

func (s *taskService) requireProjectMember(projectID, userID uint) (*model.ProjectMember, error) {
	if userID == 0 {
		return nil, ErrTaskForbidden
	}
	member, err := s.repo.GetProjectMember(projectID, userID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTaskForbidden
	}
	if err != nil {
		return nil, fmt.Errorf("query project membership: %w", err)
	}
	return member, nil
}

func (s *taskService) updateFields(task *model.Task, updates map[string]interface{}) error {
	updated, err := s.repo.UpdateFields(task.ID, task.Version, updates)
	if err != nil {
		return fmt.Errorf("update task: %w", err)
	}
	if !updated {
		return ErrTaskConflict
	}
	_ = cache.InvalidateTask(task.ID, task.ProjectID)
	return nil
}
