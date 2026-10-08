package model

import (
	"time"

	"gorm.io/gorm"
)

// Attachment 附件模型
type Attachment struct {
	gorm.Model
	ID           uint      `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID       uint      `json:"task_id" gorm:"not null;index"`      // 所属任务ID
	UploaderID   uint      `json:"uploader_id" gorm:"not null"`        // 上传者ID
	OriginalName string    `json:"original_name" gorm:"not null"`      // 原始文件名
	StoredName   string    `json:"stored_name" gorm:"not null;unique"` // 存储文件名（UUID）
	FilePath     string    `json:"file_path" gorm:"not null"`          // 磁盘路径
	FileSize     int64     `json:"file_size" gorm:"not null"`          // 文件大小（字节）
	MimeType     string    `json:"mime_type"`                          // 文件 MIME 类型
	CreatedAt    time.Time `json:"created_at"`

	// 关联关系
	Task     Task `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	Uploader User `gorm:"foreignKey:UploaderID" json:"uploader,omitempty"`
}
