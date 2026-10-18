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

// SessionRepository 作用是管理聊天会话映射的数据库操作,相当于repository层
type SessionRepository struct{ db *gorm.DB }

// NewSessionRepository 创建一个新的 SessionRepository 实例
func NewSessionRepository(db *gorm.DB) *SessionRepository { return &SessionRepository{db: db} }

// Create 作用是创建一个新的聊天会话映射,并返回该映射
func (s *SessionRepository) Create(ctx context.Context, item chat.Session, idempotencyKey string) (*chat.Session, error) {
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

// GetByPublicID 作用是根据 publicID 获取 session 映射,检查 ownerID
func (s *SessionRepository) GetByPublicID(ctx context.Context, publicID string, ownerID uint) (*chat.Session, error) {
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

// getByPublicID 作用是根据 publicID 获取 session 映射,不检查 ownerID
func (s *SessionRepository) getByPublicID(ctx context.Context, publicID string) (*chat.Session, error) {
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

// ListByOwner 作用是根据 ownerID 列出所有 session 映射,此处的ownerID 是用户ID,在表中是 owner_id 字段的值
func (s *SessionRepository) ListByOwner(ctx context.Context, ownerID uint, page, size int) ([]chat.Session, int64, error) {
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

// MarkActive 作用是将 session 状态标记为 active 并记录 agent_session_id
func (s *SessionRepository) MarkActive(ctx context.Context, publicID, agentSessionID string) (*chat.Session, error) {
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

// MarkFailed 作用是将 session 状态标记为 failed 并记录错误码
// Mark的意思是标记, 用于记录 session 创建失败的原因
func (s *SessionRepository) MarkFailed(ctx context.Context, publicID, errorCode string) error {
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

// MarkPending 作用是将 session 状态标记为 pending 并记录错误码
func (s *SessionRepository) MarkPending(ctx context.Context, publicID, errorCode string) error {
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
// toSession 作用是将 sessionRecord 转换为 chat.Session
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
