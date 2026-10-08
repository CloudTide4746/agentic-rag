package model

import "gorm.io/gorm"

// Team 团队模型
type Team struct {
	gorm.Model
	Name        string `gorm:"size:100;not null" json:"name"`
	Description string `gorm:"type:text" json:"description"`
	OwnerID     uint   `gorm:"not null" json:"owner_id"`
	Avatar      string `gorm:"size:500" json:"avatar"`

	// 关联关系
	Owner    User      `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members  []User    `gorm:"many2many:team_members;" json:"members,omitempty"`
	Projects []Project `gorm:"foreignKey:TeamID" json:"projects,omitempty"`
}
