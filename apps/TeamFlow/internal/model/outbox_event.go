package model

import "time"

type OutboxStatus string

const (
	OutboxPending    OutboxStatus = "pending"
	OutboxPublishing OutboxStatus = "publishing"
	OutboxPublished  OutboxStatus = "published"
	OutboxFailed     OutboxStatus = "failed"
)

type OutboxEvent struct {
	ID uint `gorm:"primaryKey"`

	// 与 event.Envelope.EventID 相同；同一业务事件只能写入一次 Outbox。
	EventID string `gorm:"size:64;not null;uniqueIndex"`

	EventType  string `gorm:"size:100;not null"`
	Exchange   string `gorm:"size:100;not null"`
	RoutingKey string `gorm:"size:100;not null"`

	// 完整 Envelope 的 JSON，不是只存业务 Payload。
	Payload []byte `gorm:"type:longblob;not null"`

	Status        OutboxStatus `gorm:"size:20;not null;index"`
	Attempts      int          `gorm:"not null;default:0"`
	NextAttemptAt time.Time    `gorm:"not null;index"`

	LockedBy *string `gorm:"size:100"`
	LockedAt *time.Time

	PublishedAt *time.Time
	LastError   string `gorm:"type:text"`

	CreatedAt time.Time
	UpdatedAt time.Time
}
