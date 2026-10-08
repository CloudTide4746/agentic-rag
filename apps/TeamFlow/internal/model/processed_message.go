package model

import "time"

type ProcessedMessage struct {
	EventID string `gorm:"primaryKey;size:64"`

	EventType   string    `gorm:"size:100;not null"`
	ProcessedAt time.Time `gorm:"not null"`
}

const notificationConsumerName = "notification-worker"
