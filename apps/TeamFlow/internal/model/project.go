package model

import (
	"time"

	"gorm.io/gorm"
)

// Project 项目模型
type Project struct {
	gorm.Model
	Name        string     `gorm:"size:100;not null" json:"name"`
	Description string     `gorm:"type:text" json:"description"`
	TeamID      uint       `gorm:"not null;index" json:"team_id"`
	OwnerID     uint       `gorm:"not null" json:"owner_id"`
	Status      string     `gorm:"size:20;default:'active'" json:"status"` // active/archived
	StartDate   *time.Time `json:"start_date"`
	EndDate     *time.Time `json:"end_date"`

	// 关联关系
	Team    Team   `gorm:"foreignKey:TeamID" json:"team,omitempty"`
	Owner   User   `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	Members []User `gorm:"many2many:project_members;" json:"members,omitempty"`
	Tasks   []Task `gorm:"foreignKey:ProjectID" json:"tasks,omitempty"`
}
