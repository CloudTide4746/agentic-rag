package model

import (
	"time"

	"gorm.io/gorm"
)

// Comment 评论模型
type Comment struct {
	gorm.Model
	Content   string    `gorm:"type:text;not null" json:"content"`
	TaskID    uint      `gorm:"not null;index" json:"task_id"`
	UserID    uint      `gorm:"not null;index" json:"user_id"`
	ParentID  *uint     `gorm:"index" json:"parent_id"` // 支持回复嵌套，顶层为 nil
	CreatedAt time.Time `json:"created_at"`             // 创建时间
	UpdatedAt time.Time `json:"updated_at"`             // 更新时间
	
	// 关联关系
	Task    Task      `gorm:"foreignKey:TaskID" json:"task,omitempty"`
	User    User      `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Replies []Comment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}
