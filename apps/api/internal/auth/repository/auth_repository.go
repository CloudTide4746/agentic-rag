package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"teamflow/internal/auth"
	"teamflow/internal/auth/service"
	"teamflow/internal/faults"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type userRecord struct {
	ID           uint `gorm:"primaryKey"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Username     string         `gorm:"size:50;not null;uniqueIndex"`
	Email        string         `gorm:"size:100;not null;uniqueIndex"`
	PasswordHash string         `gorm:"column:password;size:255;not null"`
	Nickname     string         `gorm:"size:50"`
	Avatar       string         `gorm:"size:500"`
	Status       int8           `gorm:"not null;default:1"`
}

func (userRecord) TableName() string { return "users" }

type UserRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) *UserRepository { return &UserRepository{db: db} }

func (r *UserRepository) Create(ctx context.Context, username, email, passwordHash string) (*auth.User, error) {
	record := userRecord{Username: username, Email: email, PasswordHash: passwordHash, Nickname: username}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, fmt.Errorf("username or email already exists: %w", faults.ErrConflict)
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	return toUser(&record), nil
}
func (r *UserRepository) ByUsername(ctx context.Context, username string) (*auth.User, string, error) {
	var record userRecord
	err := r.db.WithContext(ctx).Where("username = ? AND status = ?", username, 1).First(&record).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", fmt.Errorf("user not found: %w", faults.ErrNotFound)
	}
	if err != nil {
		return nil, "", fmt.Errorf("find user: %w", err)
	}
	return toUser(&record), record.PasswordHash, nil
}
func toUser(record *userRecord) *auth.User {
	return &auth.User{ID: strconv.FormatUint(uint64(record.ID), 10), Username: record.Username, Email: record.Email, Nickname: record.Nickname, Avatar: record.Avatar, CreatedAt: record.CreatedAt.UTC()}
}

type RedisSessions struct{ client *redis.Client }

func NewRedisSessions(client *redis.Client) *RedisSessions { return &RedisSessions{client: client} }
func (s *RedisSessions) Put(ctx context.Context, userID uint, token string, ttl time.Duration) error {
	return s.client.Set(ctx, fmt.Sprintf("auth:user:%d", userID), token, ttl).Err()
}
func (s *RedisSessions) Matches(ctx context.Context, userID uint, token string) (bool, error) {
	value, err := s.client.Get(ctx, fmt.Sprintf("auth:user:%d", userID)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == token, nil
}

var _ service.UserRepository = (*UserRepository)(nil)
var _ service.SessionStore = (*RedisSessions)(nil)
