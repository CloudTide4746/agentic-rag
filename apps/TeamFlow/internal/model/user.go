package model

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User 用户模型
type User struct {
	gorm.Model
	Username    string     `gorm:"uniqueIndex;size:50;not null" json:"username"`
	Email       string     `gorm:"uniqueIndex;size:100;not null" json:"email"`
	Password    string     `gorm:"size:255;not null" json:"-"`
	Avatar      string     `gorm:"size:500" json:"avatar"`
	Nickname    string     `gorm:"size:50" json:"nickname"`
	Status      int8       `gorm:"default:1;comment:1-正常 0-禁用" json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`

	// 关联关系
	Teams         []Team         `gorm:"many2many:team_members;" json:"teams,omitempty"`
	Projects      []Project      `gorm:"many2many:project_members;" json:"projects,omitempty"`
	Tasks         []Task         `gorm:"foreignKey:AssigneeID" json:"tasks,omitempty"`
	Notifications []Notification `gorm:"foreignKey:UserID" json:"notifications,omitempty"`
}

// BeforeCreate GORM 钩子：创建用户前自动加密密码
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashed)
	}
	if u.Nickname == "" {
		u.Nickname = u.Username
	}
	return nil
}

// BeforeUpdate GORM 钩子：更新用户前自动加密密码
// 仅当 Password 字段被显式修改时才重新哈希，避免将已哈希的密码二次加密
func (u *User) BeforeUpdate(tx *gorm.DB) error {
	if tx.Statement.Changed("Password") && u.Password != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		u.Password = string(hashed)
	}
	return nil
}
