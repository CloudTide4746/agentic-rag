package service

import (
	"context"
	"errors"
	"fmt"
	"teamflow/internal/dto/request"
	"teamflow/internal/model"
	"teamflow/storage"

	"gorm.io/gorm"
)

// TeamService 团队业务逻辑。
type TeamService struct{}

// NewTeamService 创建团队服务。
func NewTeamService() *TeamService {
	return &TeamService{}
}

// CreateTeam 创建团队，并把创建者设为 owner。
func (s *TeamService) CreateTeam(ctx context.Context, ownerID uint, req *request.CreateTeamRequest) (*model.Team, error) {
	team := &model.Team{
		Name:        req.Name,
		Description: req.Description,
		OwnerID:     ownerID,
	}

	err := storage.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(team).Error; err != nil {
			return err
		}
		return tx.Create(&model.TeamMember{
			TeamID: team.ID,
			UserID: ownerID,
			Role:   model.RoleOwner,
		}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("创建团队失败: %w", err)
	}
	return team, nil
}

// GetTeamDetail 获取团队详情，仅团队成员可查看。
func (s *TeamService) GetTeamDetail(ctx context.Context, teamID, userID uint) (*model.Team, error) {
	if _, err := s.getMember(ctx, teamID, userID); err != nil {
		return nil, err
	}

	var team model.Team
	err := storage.DB.WithContext(ctx).
		Preload("Owner").
		Preload("Members").
		First(&team, teamID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("团队不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("查询团队失败: %w", err)
	}
	return &team, nil
}

// GetTeamList 分页获取当前用户加入的团队。
func (s *TeamService) GetTeamList(ctx context.Context, userID uint, page, pageSize int) ([]model.Team, int64, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query := storage.DB.WithContext(ctx).
		Model(&model.Team{}).
		Joins("JOIN team_members ON team_members.team_id = teams.id").
		Where("team_members.user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询团队数量失败: %w", err)
	}

	var teams []model.Team
	err := query.
		Preload("Owner").
		Order("teams.id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&teams).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询团队列表失败: %w", err)
	}
	return teams, total, nil
}

// UpdateTeam 更新团队信息，仅 owner 可操作。
func (s *TeamService) UpdateTeam(ctx context.Context, teamID, userID uint, req *request.UpdateTeamRequest) (*model.Team, error) {
	team, err := s.getTeam(ctx, teamID)
	if err != nil {
		return nil, err
	}
	if team.OwnerID != userID {
		return nil, fmt.Errorf("只有团队 owner 可以更新团队")
	}
	if req.Name == nil && req.Description == nil && req.Avatar == nil {
		return nil, fmt.Errorf("至少需要提供一个更新字段")
	}

	updates := make(map[string]interface{}, 3)
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Avatar != nil {
		updates["avatar"] = *req.Avatar
	}

	if err := storage.DB.WithContext(ctx).Model(team).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新团队失败: %w", err)
	}
	return s.getTeam(ctx, teamID)
}

// DeleteTeam 解散团队，仅 owner 可操作。
func (s *TeamService) DeleteTeam(ctx context.Context, teamID, userID uint) error {
	team, err := s.getTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if team.OwnerID != userID {
		return fmt.Errorf("只有团队 owner 可以解散团队")
	}

	err = storage.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("team_id = ?", teamID).Delete(&model.TeamMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(team).Error
	})
	if err != nil {
		return fmt.Errorf("解散团队失败: %w", err)
	}
	return nil
}

// AddMember 添加成员。owner 可添加 admin/member，admin 只能添加 member。
func (s *TeamService) AddMember(ctx context.Context, teamID, operatorID uint, req *request.AddTeamMemberRequest) error {
	operator, err := s.getMember(ctx, teamID, operatorID)
	if err != nil {
		return err
	}
	if !model.HasAdminPermission(operator.Role) {
		return fmt.Errorf("只有 owner 或 admin 可以添加成员")
	}
	if operator.Role == model.RoleAdmin && req.Role != model.RoleMember {
		return fmt.Errorf("admin 只能添加普通成员")
	}

	var user model.User
	if err := storage.DB.WithContext(ctx).First(&user, req.UserID).Error; errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("用户不存在")
	} else if err != nil {
		return fmt.Errorf("查询用户失败: %w", err)
	}

	var count int64
	if err := storage.DB.WithContext(ctx).Model(&model.TeamMember{}).
		Where("team_id = ? AND user_id = ?", teamID, req.UserID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("查询团队成员失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该用户已经是团队成员")
	}

	if err := storage.DB.WithContext(ctx).Create(&model.TeamMember{
		TeamID: teamID,
		UserID: req.UserID,
		Role:   req.Role,
	}).Error; err != nil {
		return fmt.Errorf("添加成员失败: %w", err)
	}
	return nil
}

// RemoveMember 移除成员，仅 owner/admin 可操作，owner 不能被移除。
func (s *TeamService) RemoveMember(ctx context.Context, teamID, operatorID, targetUserID uint) error {
	operator, err := s.getMember(ctx, teamID, operatorID)
	if err != nil {
		return err
	}
	if !model.HasAdminPermission(operator.Role) {
		return fmt.Errorf("只有 owner 或 admin 可以移除成员")
	}

	team, err := s.getTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if team.OwnerID == targetUserID {
		return fmt.Errorf("不能移除团队 owner")
	}

	target, err := s.getMember(ctx, teamID, targetUserID)
	if err != nil {
		return err
	}
	if operator.Role == model.RoleAdmin && target.Role != model.RoleMember {
		return fmt.Errorf("admin 只能移除普通成员")
	}

	result := storage.DB.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, targetUserID).
		Delete(&model.TeamMember{})
	if result.Error != nil {
		return fmt.Errorf("移除成员失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("该用户不是团队成员")
	}
	return nil
}

// LeaveTeam 退出团队，owner 不允许直接退出。
func (s *TeamService) LeaveTeam(ctx context.Context, teamID, userID uint) error {
	team, err := s.getTeam(ctx, teamID)
	if err != nil {
		return err
	}
	if team.OwnerID == userID {
		return fmt.Errorf("owner 不能直接退出团队，请先转让所有权或解散团队")
	}
	if _, err := s.getMember(ctx, teamID, userID); err != nil {
		return err
	}

	result := storage.DB.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		Delete(&model.TeamMember{})
	if result.Error != nil {
		return fmt.Errorf("退出团队失败: %w", result.Error)
	}
	return nil
}

func (s *TeamService) getTeam(ctx context.Context, teamID uint) (*model.Team, error) {
	var team model.Team
	err := storage.DB.WithContext(ctx).First(&team, teamID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("团队不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("查询团队失败: %w", err)
	}
	return &team, nil
}

func (s *TeamService) getMember(ctx context.Context, teamID, userID uint) (*model.TeamMember, error) {
	if _, err := s.getTeam(ctx, teamID); err != nil {
		return nil, err
	}

	var member model.TeamMember
	err := storage.DB.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("你不是该团队成员")
	}
	if err != nil {
		return nil, fmt.Errorf("查询团队成员失败: %w", err)
	}
	return &member, nil
}
