package model

import "time"

// TeamMember 团队成员关联，存储额外的关联属性
type TeamMember struct {
	TeamID   uint      `gorm:"primaryKey" json:"team_id"`
	UserID   uint      `gorm:"primaryKey" json:"user_id"`
	Role     string    `gorm:"size:20;default:'member';comment:owner/admin/member" json:"role"`
	JoinedAt time.Time `gorm:"autoCreateTime" json:"joined_at"`

	// 关联
	Team Team `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}
