package model

import (
	"time"

	"gorm.io/gorm"
)

// Task 任务模型
type Task struct {
	gorm.Model
	Title       string       `gorm:"size:200;not null" json:"title"`
	Description string       `gorm:"type:text" json:"description"`
	ProjectID   uint         `gorm:"not null;index" json:"project_id"`
	AssigneeID  *uint        `gorm:"index" json:"assignee_id"` // 可为空（未分配）
	CreatorID   uint         `gorm:"not null" json:"creator_id"`
	Status      TaskStatus   `gorm:"size:20;default:'todo'" json:"status"` // todo/doing/done
	Priority    TaskPriority `gorm:"size:20;default:'medium'" json:"priority"`
	DueDate     *time.Time   `json:"due_date"`
	Version     int          `gorm:"default:0" json:"version"`

	// 关联关系
	Project     Project      `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	Assignee    *User        `gorm:"foreignKey:AssigneeID" json:"assignee,omitempty"`
	Creator     User         `gorm:"foreignKey:CreatorID" json:"creator,omitempty"`
	Comments    []Comment    `gorm:"foreignKey:TaskID" json:"comments,omitempty"`
	Attachments []Attachment `gorm:"foreignKey:TaskID" json:"attachments,omitempty"`
}
type TaskStatus string

const (
	TaskStatusTodo       TaskStatus = "todo"
	TaskStatusInProgress TaskStatus = "in_progress"
	TaskStatusReview     TaskStatus = "review"
	TaskStatusDone       TaskStatus = "done"
)

type TaskPriority string

const (
	TaskPriorityLow    TaskPriority = "low"
	TaskPriorityMedium TaskPriority = "medium"
	TaskPriorityHigh   TaskPriority = "high"
	TaskPriorityNone   TaskPriority = "none"
)

// IsValidTaskStatus 校验任务状态是否有效
func IsValidTaskStatus(status TaskStatus) bool {
	switch status {
	case TaskStatusTodo, TaskStatusInProgress, TaskStatusReview, TaskStatusDone:
		return true
	default:
		return false
	}
}

// IsValidTaskPriority 校验任务优先级是否有效
func IsValidTaskPriority(priority TaskPriority) bool {
	switch priority {
	case TaskPriorityLow, TaskPriorityMedium, TaskPriorityHigh, TaskPriorityNone:
		return true
	default:
		return false
	}
}

func (Task) TableName() string {
	return "tasks"
}
