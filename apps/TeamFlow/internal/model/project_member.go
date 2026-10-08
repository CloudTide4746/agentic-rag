package model

import "time"

// ProjectMember 项目成员关联，存储额外的关联属性
type ProjectMember struct {
	ProjectID uint      `gorm:"primaryKey" json:"project_id"`
	UserID    uint      `gorm:"primaryKey" json:"user_id"`
	Role      string    `gorm:"size:20;default:'member'" json:"role"` // owner/admin/member
	JoinedAt  time.Time `gorm:"autoCreateTime" json:"joined_at"`

	Project Project `gorm:"foreignKey:ProjectID" json:"project,omitempty"`
	User    User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
