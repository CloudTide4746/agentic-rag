package service

import (
	"errors"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"teamflow/internal/model"
	"teamflow/internal/repository"
	"teamflow/pkg/utils"
)

var (
	ErrAttachmentNotFound  = errors.New("附件不存在")
	ErrAttachmentForbidden = errors.New("无权操作此附件")
)

type AttachmentServiceInterface interface {
	UploadAttachment(taskID, uploaderID uint, header *multipart.FileHeader) (*model.Attachment, error)
	GetAttachments(taskID uint) ([]*model.Attachment, error)
	DeleteAttachment(attachmentID, operatorID uint, isAdmin bool) error
	GetByID(attachmentID uint) (*model.Attachment, error)
}
type AttachmentService struct {
	repo repository.AttachmentRepository
}

func NewAttachmentService(repo repository.AttachmentRepository) *AttachmentService {
	return &AttachmentService{repo: repo}
}

func (s *AttachmentService) UploadAttachment(taskID, uploaderID uint, header *multipart.FileHeader) (*model.Attachment, error) {
	// 验证文件是否符合要求
	ext, err := utils.ValidateFile(header)
	if err != nil {
		return nil, err
	}
	// 生成文件名
	storedName := utils.GenerateUniqueFileName(ext)
	uploadDir := utils.GetUploadPath()
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, errors.New("创建上传目录失败")
	}

	src, err := header.Open()
	if err != nil {
		return nil, err
	}
	defer func(src multipart.File) {
		err := src.Close()
		if err != nil {

		}
	}(src)
	// 文件操作
	savePath := filepath.Join(uploadDir, storedName)
	dst, err := os.Create(savePath)
	if err != nil {
		return nil, err
	}
	defer func(dst io.Closer) {
		err := dst.Close()
		if err != nil {

		}
	}(dst)

	if _, err := dst.ReadFrom(src); err != nil {
		// 读取文件失败
		// 清理已创建的文件
		if err := os.Remove(savePath); err != nil {
			return nil, err
		}
		return nil, err
	}

	// 数据库操作
	attachment := &model.Attachment{
		TaskID:       taskID,
		UploaderID:   uploaderID,
		OriginalName: header.Filename,
		StoredName:   storedName,
		FilePath:     savePath,
		FileSize:     header.Size,
		MimeType:     header.Header.Get("Content-Type"),
	}
	err = s.repo.Create(attachment)
	if err != nil {
		// 创建附件失败
		// 清理已创建的文件
		if err := os.Remove(savePath); err != nil {
			return nil, err
		}
		return nil, err
	}
	return attachment, nil
}

// GetAttachments 获取任务下的所有附件
func (s *AttachmentService) GetAttachments(taskID uint) ([]*model.Attachment, error) {
	var attachments []*model.Attachment
	attachments, err := s.repo.FindByTaskID(taskID)
	// 检查查询结果是否为空
	if err != nil {
		return nil, err
	}
	return attachments, nil
}

// DeleteAttachment 删除附件
func (s *AttachmentService) DeleteAttachment(attachmentID, operatorID uint, isAdmin bool) error {
	attachment, err := s.repo.FindByID(attachmentID)
	if err != nil {
		return ErrAttachmentNotFound
	}

	// 权限校验：管理员或上传者本人可以删除
	if !isAdmin && attachment.UploaderID != operatorID {
		return ErrAttachmentForbidden
	}

	// 先从数据库删除，再删除磁盘文件（顺序很重要）
	if err := s.repo.DeleteByID(attachmentID); err != nil {
		return err
	}
	os.Remove(attachment.FilePath) // 磁盘删除失败不影响接口响应

	return nil
}
func (s *AttachmentService) GetByID(attachmentID uint) (*model.Attachment, error) {
	return s.repo.FindByID(attachmentID)
}
