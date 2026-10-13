package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"teamflow/internal/faults"
	"teamflow/internal/task"

	"gorm.io/gorm"
)

type taskRecord struct {
	ID                   uint `gorm:"primaryKey"`
	CreatedAt, UpdatedAt time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
	Title                string         `gorm:"size:200;not null"`
	Description          string         `gorm:"type:text"`
	ProjectID            uint           `gorm:"not null;index"`
	AssigneeID           *uint          `gorm:"index"`
	CreatorID            uint           `gorm:"not null"`
	Status               string         `gorm:"size:20;not null"`
	Priority             string         `gorm:"size:20;not null"`
	DueDate              *time.Time
	Version              int `gorm:"not null;default:0"`
}

func (taskRecord) TableName() string { return "tasks" }

type TaskRepository struct{ db *gorm.DB }
// NewTaskRepository 创建任务仓库
func NewTaskRepository(db *gorm.DB) *TaskRepository { return &TaskRepository{db: db} }
// Create 创建任务
func (r *TaskRepository) Create(ctx context.Context, item task.Task) (*task.Task, error) {
	pid, _ := strconv.ParseUint(item.ProjectID, 10, 64)
	cid, _ := strconv.ParseUint(item.CreatorID, 10, 64)
	row := taskRecord{Title: item.Title, Description: item.Description, ProjectID: uint(pid), CreatorID: uint(cid), Status: item.Status, Priority: item.Priority, DueDate: item.DueDate}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return toTask(&row), nil
}	
// ListByProject 获取项目任务列表
func (r *TaskRepository) ListByProject(ctx context.Context, projectID uint, page, size int) ([]task.Task, int64, error) {
	q := r.db.WithContext(ctx).Model(&taskRecord{}).Where("project_id = ?", projectID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tasks: %w", err)
	}
	var rows []taskRecord
	if err := q.Order("created_at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list tasks: %w", err)
	}
	items := make([]task.Task, 0, len(rows))
	for i := range rows {
		items = append(items, *toTask(&rows[i]))
	}
	return items, total, nil
}
// Get 获取任务
func (r *TaskRepository) Get(ctx context.Context, id uint) (*task.Task, error) {
	var row taskRecord
	err := r.db.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, faults.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	return toTask(&row), nil
}
// Update 更新任务
func (r *TaskRepository) Update(ctx context.Context, id uint, expectedVersion int, fields map[string]any) (*task.Task, error) {
	var current taskRecord
	if err := r.db.WithContext(ctx).First(&current, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, faults.ErrNotFound
		}
		return nil, fmt.Errorf("read task before update: %w", err)
	}
	if current.Version != expectedVersion {
		return nil, faults.ErrConflict
	}
	fields["version"] = expectedVersion + 1
	res := r.db.WithContext(ctx).Model(&taskRecord{}).Where("id = ? AND version = ?", id, expectedVersion).Updates(fields)
	if res.Error != nil {
		return nil, fmt.Errorf("update task: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, faults.ErrConflict
	}
	return r.Get(ctx, id)
}
// Delete 删除任务
func (r *TaskRepository) Delete(ctx context.Context, id uint) error {
	res := r.db.WithContext(ctx).Delete(&taskRecord{}, id)
	if res.Error != nil {
		return fmt.Errorf("delete task: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return faults.ErrNotFound
	}
	return nil
}
// toTask 转换任务记录为任务
func toTask(row *taskRecord) *task.Task {
	var assignee *string
	if row.AssigneeID != nil {
		v := strconv.FormatUint(uint64(*row.AssigneeID), 10)
		assignee = &v
	}
	var dueDate *time.Time
	if row.DueDate != nil {
		value := row.DueDate.UTC()
		dueDate = &value
	}
	return &task.Task{ID: strconv.FormatUint(uint64(row.ID), 10), Title: row.Title, Description: row.Description, ProjectID: strconv.FormatUint(uint64(row.ProjectID), 10), AssigneeID: assignee, CreatorID: strconv.FormatUint(uint64(row.CreatorID), 10), Status: row.Status, Priority: row.Priority, DueDate: dueDate, Version: row.Version, CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC()}
}
