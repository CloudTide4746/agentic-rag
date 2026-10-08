package database

import (
	"teamflow/internal/model"
	"teamflow/storage"
)

// AutoMigrate 自动迁移所有表结构
func AutoMigrate() error {
	return storage.DB.AutoMigrate(
		&model.User{},
		&model.Team{},
		&model.TeamMember{},
		&model.Project{},
		&model.ProjectMember{},
		&model.Task{},
		&model.Comment{},
		&model.Attachment{},
		&model.Notification{},
		&model.ProcessedMessage{},
		&model.OutboxEvent{},
	)
}
