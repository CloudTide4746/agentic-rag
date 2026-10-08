package repository

import (
	"teamflow/internal/model"

	"gorm.io/gorm"
)

type AttachmentRepository interface {
	Create(attachment *model.Attachment) error
	FindByTaskID(taskID uint) ([]*model.Attachment, error)
	FindByID(id uint) (*model.Attachment, error) //通过用户ID查询附件详情
	DeleteByID(id uint) error                    //通过用户ID删除
}
type attachmentRepository struct {
	db *gorm.DB
}

func NewAttachmentRepository(db *gorm.DB) AttachmentRepository {
	return &attachmentRepository{db: db}
}

// Create 创建附件
func (r *attachmentRepository) Create(attachment *model.Attachment) error {
	return r.db.Create(attachment).Error
}

// FindByTaskID 根据任务ID查询附件
func (r *attachmentRepository) FindByTaskID(taskID uint) ([]*model.Attachment, error) {
	if taskID == 0 {
		return nil, nil
	}
	var attachments []*model.Attachment
	err := r.db.Where("task_id = ?", taskID).Find(&attachments).Error
	return attachments, err
}

// FindByID 根据ID查询附件
func (r *attachmentRepository) FindByID(id uint) (*model.Attachment, error) {
	if id == 0 {
		return nil, nil
	}
	var attachment model.Attachment
	err := r.db.Where("id = ?", id).First(&attachment).Error
	return &attachment, err
}

// DeleteByID 根据ID删除附件
func (r *attachmentRepository) DeleteByID(id uint) error {
	if id == 0 {
		return nil
	}
	err := r.db.Delete(&model.Attachment{}, id).Error
	return err
}
