package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"teamflow/internal/infrastructure/faults"
	"teamflow/internal/team"
	"teamflow/internal/team/service"

	"gorm.io/gorm"
)

type teamRecord struct {
	ID                   uint `gorm:"primaryKey"`
	CreatedAt, UpdatedAt time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
	Name                 string         `gorm:"size:100;not null"`
	Description          string         `gorm:"type:text"`
	OwnerID              uint           `gorm:"not null;index"`
	Avatar               string         `gorm:"size:500"`
}

// TableName 返回团队记录对应的数据库表名。
func (teamRecord) TableName() string { return "teams" }

type memberRecord struct {
	TeamID   uint      `gorm:"primaryKey"`
	UserID   uint      `gorm:"primaryKey"`
	Role     string    `gorm:"size:20;not null"`
	JoinedAt time.Time `gorm:"autoCreateTime"`
}

// TableName 返回团队成员记录对应的数据库表名。
func (memberRecord) TableName() string { return "team_members" }

type TeamRepository struct{ db *gorm.DB }

var _ service.Repository = (*TeamRepository)(nil)

// NewTeamRepository 创建使用指定数据库连接的团队仓库。
func NewTeamRepository(db *gorm.DB) *TeamRepository { return &TeamRepository{db: db} }

// Create 在同一事务中创建团队及所有者的团队成员关系。
func (r *TeamRepository) Create(ctx context.Context, item team.Team) (*team.Team, error) {
	ownerID, err := strconv.ParseUint(item.OwnerID, 10, 64)
	if err != nil || ownerID == 0 {
		return nil, fmt.Errorf("invalid owner id: %w", faults.ErrValidation)
	}
	record := teamRecord{Name: item.Name, Description: item.Description, OwnerID: uint(ownerID), Avatar: item.Avatar}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		return tx.Create(&memberRecord{TeamID: record.ID, UserID: uint(ownerID), Role: "owner"}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("create team: %w", err)
	}
	return toTeam(&record), nil
}

// ListForUser 分页查询用户所属团队，并返回符合条件的团队总数。
func (r *TeamRepository) ListForUser(ctx context.Context, userID uint, page, size int) ([]team.Team, int64, error) {
	query := r.db.WithContext(ctx).Model(&teamRecord{}).Joins("JOIN team_members ON team_members.team_id = teams.id").Where("team_members.user_id = ?", userID)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count teams: %w", err)
	}
	var rows []teamRecord
	if err := query.Order("teams.id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list teams: %w", err)
	}
	items := make([]team.Team, 0, len(rows))
	for i := range rows {
		items = append(items, *toTeam(&rows[i]))
	}
	return items, total, nil
}

// Get 按 ID 查询团队，记录不存在时返回未找到错误。
func (r *TeamRepository) Get(ctx context.Context, id uint) (*team.Team, error) {
	var row teamRecord
	err := r.db.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, faults.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get team: %w", err)
	}
	return toTeam(&row), nil
}

// IsMember 检查用户是否存在指定团队的成员关系。
func (r *TeamRepository) IsMember(ctx context.Context, teamID, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&memberRecord{}).Where("team_id = ? AND user_id = ?", teamID, userID).Count(&count).Error
	return count > 0, err
}

// Update 更新指定团队的字段并返回最新团队信息。
func (r *TeamRepository) Update(ctx context.Context, id uint, fields map[string]any) (*team.Team, error) {
	result := r.db.WithContext(ctx).Model(&teamRecord{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return nil, fmt.Errorf("update team: %w", result.Error)
	}
	return r.Get(ctx, id)
}

// Delete 在同一事务中删除团队成员关系并软删除团队。
func (r *TeamRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("team_id = ?", id).Delete(&memberRecord{}).Error; err != nil {
			return fmt.Errorf("delete team memberships: %w", err)
		}
		result := tx.Delete(&teamRecord{}, id)
		if result.Error != nil {
			return fmt.Errorf("delete team: %w", result.Error)
		}
		if result.RowsAffected == 0 {
			return faults.ErrNotFound
		}
		return nil
	})
}

// toTeam 将数据库团队记录转换为团队信息，并统一 ID 和 UTC 时间格式。
func toTeam(row *teamRecord) *team.Team {
	return &team.Team{ID: strconv.FormatUint(uint64(row.ID), 10), Name: row.Name, Description: row.Description, OwnerID: strconv.FormatUint(uint64(row.OwnerID), 10), Avatar: row.Avatar, CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC()}
}
