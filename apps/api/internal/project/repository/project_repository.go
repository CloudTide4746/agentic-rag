package repository

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"teamflow/internal/faults"
	"teamflow/internal/project"
	"teamflow/internal/project/service"

	"gorm.io/gorm"
)

type projectRecord struct {
	ID                   uint `gorm:"primaryKey"`
	CreatedAt, UpdatedAt time.Time
	DeletedAt            gorm.DeletedAt `gorm:"index"`
	Name                 string         `gorm:"size:100;not null"`
	Description          string         `gorm:"type:text"`
	TeamID               uint           `gorm:"not null;index"`
	OwnerID              uint           `gorm:"not null;index"`
	Status               string         `gorm:"size:20;not null"`
	StartDate, EndDate   *time.Time
}

func (projectRecord) TableName() string { return "projects" }

type memberRecord struct {
	ProjectID uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"primaryKey"`
	Role      string    `gorm:"size:20;not null"`
	JoinedAt  time.Time `gorm:"autoCreateTime"`
}

func (memberRecord) TableName() string { return "project_members" }

type ProjectRepository struct{ db *gorm.DB }

func NewProjectRepository(db *gorm.DB) *ProjectRepository { return &ProjectRepository{db: db} }
func (r *ProjectRepository) Create(ctx context.Context, item project.Project) (*project.Project, error) {
	teamID, _ := strconv.ParseUint(item.TeamID, 10, 64)
	ownerID, _ := strconv.ParseUint(item.OwnerID, 10, 64)
	row := projectRecord{Name: item.Name, Description: item.Description, TeamID: uint(teamID), OwnerID: uint(ownerID), Status: item.Status, StartDate: item.StartDate, EndDate: item.EndDate}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&row).Error; err != nil {
			return err
		}
		return tx.Create(&memberRecord{ProjectID: row.ID, UserID: uint(ownerID), Role: "owner"}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("create project: %w", err)
	}
	return toProject(&row), nil
}
func (r *ProjectRepository) ListByTeam(ctx context.Context, teamID uint, page, size int) ([]project.Project, int64, error) {
	q := r.db.WithContext(ctx).Model(&projectRecord{}).Where("team_id = ?", teamID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var rows []projectRecord
	if err := q.Order("id DESC").Offset((page - 1) * size).Limit(size).Find(&rows).Error; err != nil {
		return nil, 0, fmt.Errorf("list projects: %w", err)
	}
	items := make([]project.Project, 0, len(rows))
	for i := range rows {
		items = append(items, *toProject(&rows[i]))
	}
	return items, total, nil
}
func (r *ProjectRepository) Get(ctx context.Context, id uint) (*project.Project, error) {
	var row projectRecord
	err := r.db.WithContext(ctx).First(&row, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, faults.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get project: %w", err)
	}
	return toProject(&row), nil
}
func (r *ProjectRepository) IsMember(ctx context.Context, projectID, userID uint) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&memberRecord{}).Where("project_id = ? AND user_id = ?", projectID, userID).Count(&count).Error
	return count > 0, err
}
func (r *ProjectRepository) Update(ctx context.Context, id uint, fields map[string]any) (*project.Project, error) {
	res := r.db.WithContext(ctx).Model(&projectRecord{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return nil, fmt.Errorf("update project: %w", res.Error)
	}
	return r.Get(ctx, id)
}
func (r *ProjectRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", id).Delete(&memberRecord{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&projectRecord{}, id)
		if res.Error != nil {
			return fmt.Errorf("delete project: %w", res.Error)
		}
		if res.RowsAffected == 0 {
			return faults.ErrNotFound
		}
		return nil
	})
}
func toProject(row *projectRecord) *project.Project {
	var startDate, endDate *time.Time
	if row.StartDate != nil {
		value := row.StartDate.UTC()
		startDate = &value
	}
	if row.EndDate != nil {
		value := row.EndDate.UTC()
		endDate = &value
	}
	return &project.Project{ID: strconv.FormatUint(uint64(row.ID), 10), Name: row.Name, Description: row.Description, TeamID: strconv.FormatUint(uint64(row.TeamID), 10), OwnerID: strconv.FormatUint(uint64(row.OwnerID), 10), Status: row.Status, StartDate: startDate, EndDate: endDate, CreatedAt: row.CreatedAt.UTC(), UpdatedAt: row.UpdatedAt.UTC()}
}

var _ service.Repository = (*ProjectRepository)(nil)
