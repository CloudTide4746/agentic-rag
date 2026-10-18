package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"teamflow/internal/chat"
	"teamflow/internal/faults"

	"gorm.io/gorm"
)

type sessionRecord struct {
	ID             uint `gorm:"primaryKey"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
	DeletedAt      gorm.DeletedAt `gorm:"index"`
	PublicID       string         `gorm:"size:64;not null;uniqueIndex"`
	OwnerID        uint           `gorm:"not null;index"`
	AgentSessionID *string        `gorm:"size:64"`
	Title          string         `gorm:"size:128;not null"`
	Status         string         `gorm:"size:20;not null"`
	LastErrorCode  *string        `gorm:"size:64"`
	IdempotencyKey string         `gorm:"size:64;not null;uniqueIndex"`
}

func (sessionRecord) TableName() string { return "chat_platform_sessions" }

// SessionStore owns the platform-side mapping table. It never touches the
// Agent's chat_session / chat_message tables (ADR-0003).
type SessionStore struct{ db *gorm.DB }

func NewSessionStore(db *gorm.DB) *SessionStore { return &SessionStore{db: db} }

func (s *SessionStore) Create(ctx context.Context, item chat.Session, idempotencyKey string) (*chat.Session, error) {
	row := sessionRecord{
		PublicID:       item.PublicID,
		OwnerID:        parseID(item.OwnerID),
		Title:          item.Title,
		Status:         item.Status,
		IdempotencyKey: idempotencyKey,
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("create chat session mapping: %w", err)
	}
	return toSession(&row), nil
}

// GetByPublicID returns the mapping only when it belongs to ownerID, so a
// caller can never reach another user's session by guessing its public id.
func (s *SessionStore) GetByPublicID(ctx context.Context, publicID string, ownerID uint) (*chat.Session, error) {
	var row sessionRecord
	err := s.db.WithContext(ctx).Where("public_id = ? AND owner_id = ?", publicID, ownerID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, faults.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get chat session mapping: %w", err)
	}
	return toSession(&row), nil
}

// getByPublicID reads a mapping without an owner filter. It is unexported so
// the authorization-scoped read stays the only public entry point.
func (s *SessionStore) getByPublicID(ctx context.Context, publicID string) (*chat.Session, error) {
	var row sessionRecord
	err := s.db.WithContext(ctx).Where("public_id = ?", publicID).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, faults.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get chat session mapping: %w", err)
	}
	return toSession(&row), nil
}

func (s *SessionStore) ListByOwner(ctx context.Context, ownerID uint, page, size int) ([]chat.Session, int64, error) {
	q := s.db.WithContext(ctx).Model(&sessionRecord{}).Where("owner_id = ?", ownerID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count chat sessions: %w", err)
	}
	var rows []sessionRecord
	if err := q.Order("updated_at DESC, id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list chat sessions: %w", err)
	}
	items := make([]chat.Session, 0, len(rows))
	for i := range rows {
		items = append(items, *toSession(&rows[i]))
	}
	return items, total, nil
}

// MarkActive records a confirmed Agent session id after creation succeeded and
// returns the updated mapping so the caller can report the committed state.
func (s *SessionStore) MarkActive(ctx context.Context, publicID, agentSessionID string) (*chat.Session, error) {
	res := s.db.WithContext(ctx).Model(&sessionRecord{}).
		Where("public_id = ?", publicID).
		Updates(map[string]any{
			"agent_session_id": agentSessionID,
			"status":           chat.StatusActive,
			"last_error_code":  nil,
			"updated_at":       time.Now().UTC(),
		})
	if res.Error != nil {
		return nil, fmt.Errorf("mark chat session active: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, faults.ErrNotFound
	}
	return s.getByPublicID(ctx, publicID)
}

// MarkFailed records an explicit Agent rejection. The mapping is kept so the
// caller can inspect or retry it instead of silently losing the attempt.
func (s *SessionStore) MarkFailed(ctx context.Context, publicID, errorCode string) error {
	res := s.db.WithContext(ctx).Model(&sessionRecord{}).
		Where("public_id = ?", publicID).
		Updates(map[string]any{"status": chat.StatusFailed, "last_error_code": errorCode})
	if res.Error != nil {
		return fmt.Errorf("mark chat session failed: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return faults.ErrNotFound
	}
	return nil
}

// MarkPending records an unknown creation outcome (timeout or lost response).
// The record is reconciled later by idempotency key; it is never retried
// blindly, because that could create a second Agent session (ADR-0003).
func (s *SessionStore) MarkPending(ctx context.Context, publicID, errorCode string) error {
	res := s.db.WithContext(ctx).Model(&sessionRecord{}).
		Where("public_id = ?", publicID).
		Updates(map[string]any{"status": chat.StatusPending, "last_error_code": errorCode})
	if res.Error != nil {
		return fmt.Errorf("mark chat session pending: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return faults.ErrNotFound
	}
	return nil
}

func toSession(row *sessionRecord) *chat.Session {
	item := &chat.Session{
		PublicID:  row.PublicID,
		Title:     row.Title,
		Status:    row.Status,
		OwnerID:   strconv.FormatUint(uint64(row.OwnerID), 10),
		CreatedAt: row.CreatedAt.UTC(),
		UpdatedAt: row.UpdatedAt.UTC(),
	}
	if row.AgentSessionID != nil {
		item.AgentSessionID = *row.AgentSessionID
	}
	if row.LastErrorCode != nil {
		item.LastErrorCode = *row.LastErrorCode
	}
	return item
}

func parseID(value string) uint {
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0
	}
	return uint(parsed)
}
