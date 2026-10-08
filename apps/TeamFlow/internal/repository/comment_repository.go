package repository

import (
	"teamflow/internal/model"

	"gorm.io/gorm"
)

type CommentRepository interface {
	Create(comment *model.Comment) error
	FindByTaskID(taskID uint, page, size int) ([]*model.Comment, int64, error)
	FindByID(id uint) (*model.Comment, error)
	Delete(id uint) error
}
type commentRepository struct {
	db *gorm.DB
}

func NewCommentRepository(db *gorm.DB) CommentRepository {
	return &commentRepository{db: db}
}

// Create 创建评论
// 返回值：错误
func (r *commentRepository) Create(comment *model.Comment) error {
	return r.db.Create(comment).Error
}

// FindByTaskID 按任务ID分页查询评论，同时预加载用户信息
// 返回值：评论列表、总数量、错误
func (r *commentRepository) FindByTaskID(taskID uint, page, pageSize int) ([]*model.Comment, int64, error) {
	var comments []*model.Comment
	var total int64

	// 先统计总数
	if err := r.db.Model(&model.Comment{}).
		Where("task_id = ?", taskID).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询，预加载评论者信息（只加载必要字段）
	offset := (page - 1) * pageSize
	if err := r.db.
		Preload("User", func(db *gorm.DB) *gorm.DB {
			return db.Select("id, username, avatar")
		}).
		Where("task_id = ?", taskID).
		Order("created_at ASC"). // 按时间正序，便于阅读
		Limit(pageSize).
		Offset(offset).
		Find(&comments).Error; err != nil {
		return nil, 0, err
	}

	return comments, total, nil
}

// FindByID 按ID查询评论
// 返回值：评论、错误
func (r *commentRepository) FindByID(id uint) (*model.Comment, error) {
	var comment model.Comment
	err := r.db.First(&comment, id).Error
	if err != nil {
		return nil, err
	}
	return &comment, nil
}

// Delete 删除评论
func (r *commentRepository) Delete(id uint) error {
	return r.db.Delete(&model.Comment{}, id).Error
}
