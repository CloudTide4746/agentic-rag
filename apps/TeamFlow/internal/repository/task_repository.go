package repository

import (
	"teamflow/internal/dto"
	"teamflow/internal/model"

	"gorm.io/gorm"
)

type TaskRepository interface {
	Create(task *model.Task) error
	GetByID(id uint) (*model.Task, error)
	Update(task *model.Task) error
	UpdateFields(id uint, version int, updates map[string]interface{}) (bool, error)
	// UpdateFields 更新任务字段

	UpdateFieldsTx(tx *gorm.DB, id uint, version int, updates map[string]interface{}) (bool, error)
	Delete(id uint) error
	List(projectID uint, page, size int) ([]*model.Task, int64, error)
	GetProjectMember(projectID, userID uint) (*model.ProjectMember, error)
	ListTasks(query dto.TaskQuery) ([]*model.Task, int64, error)
}
