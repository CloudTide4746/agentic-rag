package repository

import (
	"teamflow/internal/dto"
	"teamflow/internal/model"
	"time"

	"gorm.io/gorm"
)

type taskRepository struct {
	db *gorm.DB
}

func (r *taskRepository) ListTask(query dto.TaskQuery) ([]*model.Task, int64, error) {
	//TODO implement me
	panic("implement me")
}

func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

func (r *taskRepository) Create(task *model.Task) error {
	return r.db.Create(task).Error
}

func (r *taskRepository) GetByID(id uint) (*model.Task, error) {
	var task model.Task
	err := r.db.First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (r *taskRepository) Update(task *model.Task) error {
	return r.db.Save(task).Error
}

func (r *taskRepository) UpdateFields(id uint, version int, updates map[string]interface{}) (bool, error) {
	return r.UpdateFieldsTx(r.db, id, version, updates)
}

// UpdateFieldsTx 更新任务字段
func (r *taskRepository) UpdateFieldsTx(tx *gorm.DB, id uint, version int, updates map[string]interface{}) (bool, error) {
	updates["version"] = gorm.Expr("version + 1")
	result := tx.Model(&model.Task{}).
		Where("id = ? AND version = ?", id, version).
		Updates(updates)	
	if result.Error != nil {
		return false, result.Error
	}
	return result.RowsAffected == 1, nil
}

func (r *taskRepository) Delete(id uint) error {
	return r.db.Delete(&model.Task{}, id).Error
}

func (r *taskRepository) List(projectID uint, page, size int) ([]*model.Task, int64, error) {
	var tasks []*model.Task
	var count int64
	query := r.db.Model(&model.Task{}).Where("project_id = ?", projectID)
	if err := query.Count(&count).Error; err != nil {
		return nil, 0, err
	}
	err := query.
		Order("created_at DESC, id DESC").
		Offset((page - 1) * size).
		Limit(size).
		Find(&tasks).Error
	if err != nil {
		return nil, 0, err
	}
	return tasks, count, nil
}

func (r *taskRepository) GetProjectMember(projectID, userID uint) (*model.ProjectMember, error) {
	var member model.ProjectMember
	err := r.db.
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&member).Error
	if err != nil {
		return nil, err
	}
	return &member, nil
}

// ListTasks 根据筛选条件查询任务列表
func (r *taskRepository) ListTasks(query dto.TaskQuery) ([]*model.Task, int64, error) {
	db := r.db.Model(&model.Task{})

	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Priority != "" {
		db = db.Where("priority = ?", query.Priority)
	}
	if query.AssigneeID != 0 {
		db = db.Where("assignee_id = ?", query.AssigneeID)
	}
	if query.DueDateStart != "" {
		db = db.Where("due_date >= ?", query.DueDateStart)
	}
	if query.DueDateEnd != "" {
		end, err := time.Parse("2006-01-02", query.DueDateEnd)
		if err != nil {
			return nil, 0, err
		}
		db = db.Where("due_date <= ?", end)
	}

	var totalCount int64
	var tasks []*model.Task

	// 排序
	orderClause := buildOrderClause(query.SortBy, query.SortDir)
	db = db.Order(orderClause)

	// 分页
	offset := (query.Page - 1) * query.PageSize
	db = db.Offset(offset).Limit(query.PageSize)

	// 执行查询
	if err := db.Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	// 统计总记录数
	if err := db.Count(&totalCount).Error; err != nil {
		return nil, 0, err
	}
	return tasks, totalCount, nil
}

// 允许排序的字段白名单
var allowedSortFields = map[string]string{
	"created_at": "created_at",
	"priority":   "priority",
	"due_date":   "due_date",
}

// 允许的排序方向白名单
var allowedSortDirs = map[string]string{
	"asc":  "ASC",
	"desc": "DESC",
}

// buildOrderClause 构建安全的 ORDER BY 子句
// 非法字段默认按 created_at DESC 排序
func buildOrderClause(sortBy, sortDir string) string {
	field, ok := allowedSortFields[sortBy]
	if !ok {
		field = "created_at" // 默认字段
	}

	dir, ok := allowedSortDirs[sortDir]
	if !ok {
		dir = "DESC" // 默认方向
	}

	return field + " " + dir
}
