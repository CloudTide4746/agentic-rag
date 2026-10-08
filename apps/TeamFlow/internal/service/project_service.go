package service

import (
	"context"
	"errors"
	"fmt"
	"teamflow/internal/dto/request"
	"teamflow/internal/model"
	"teamflow/storage"
	"time"

	"gorm.io/gorm"
)

// ProjectService 项目业务逻辑。
type ProjectService struct{}

// ProjectStats 项目统计。
type ProjectStats struct {
	MemberCount int64 `json:"member_count"`
	TaskCount   int64 `json:"task_count"`
	TodoCount   int64 `json:"todo_count"`
	DoingCount  int64 `json:"doing_count"`
	DoneCount   int64 `json:"done_count"`
}

func NewProjectService() *ProjectService {
	return &ProjectService{}
}

// CreateProject 创建项目，并将创建者设为项目 owner。
func (s *ProjectService) CreateProject(ctx context.Context, teamID, ownerID uint, req *request.CreateProjectRequest) (*model.Project, error) {
	if _, err := s.getTeamMember(ctx, teamID, ownerID); err != nil {
		return nil, err
	}
	if err := validateProjectDates(req.StartDate, req.EndDate); err != nil {
		return nil, err
	}

	project := &model.Project{
		Name:        req.Name,
		Description: req.Description,
		TeamID:      teamID,
		OwnerID:     ownerID,
		Status:      "active",
		StartDate:   req.StartDate,
		EndDate:     req.EndDate,
	}

	err := storage.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(project).Error; err != nil {
			return err
		}
		return tx.Create(&model.ProjectMember{
			ProjectID: project.ID,
			UserID:    ownerID,
			Role:      model.RoleOwner,
		}).Error
	})
	if err != nil {
		return nil, fmt.Errorf("创建项目失败: %w", err)
	}
	return project, nil
}

// GetProjectList 分页获取团队项目。
func (s *ProjectService) GetProjectList(ctx context.Context, teamID, userID uint, page, pageSize int) ([]model.Project, int64, error) {
	if _, err := s.getTeamMember(ctx, teamID, userID); err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	query := storage.DB.WithContext(ctx).Model(&model.Project{}).Where("team_id = ?", teamID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("查询项目数量失败: %w", err)
	}

	var projects []model.Project
	err := query.
		Preload("Owner").
		Order("id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&projects).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询项目列表失败: %w", err)
	}
	return projects, total, nil
}

// GetProjectDetail 获取项目详情，仅项目成员可查看。
func (s *ProjectService) GetProjectDetail(ctx context.Context, projectID, userID uint) (*model.Project, error) {
	if _, err := s.getProjectMember(ctx, projectID, userID); err != nil {
		return nil, err
	}

	var project model.Project
	err := storage.DB.WithContext(ctx).
		Preload("Team").
		Preload("Owner").
		Preload("Members").
		First(&project, projectID).Error
	if err != nil {
		return nil, fmt.Errorf("查询项目详情失败: %w", err)
	}
	return &project, nil
}

// UpdateProject 更新项目，仅项目 owner 可操作。
func (s *ProjectService) UpdateProject(ctx context.Context, projectID, userID uint, req *request.UpdateProjectRequest) (*model.Project, error) {
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return nil, err
	}
	if project.OwnerID != userID {
		return nil, fmt.Errorf("只有项目 owner 可以更新项目")
	}
	if req.Name == nil && req.Description == nil && req.Status == nil && req.StartDate == nil && req.EndDate == nil {
		return nil, fmt.Errorf("至少需要提供一个更新字段")
	}

	startDate := project.StartDate
	endDate := project.EndDate
	if req.StartDate != nil {
		startDate = req.StartDate
	}
	if req.EndDate != nil {
		endDate = req.EndDate
	}
	if err := validateProjectDates(startDate, endDate); err != nil {
		return nil, err
	}

	updates := make(map[string]interface{}, 5)
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Description != nil {
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	if req.StartDate != nil {
		updates["start_date"] = *req.StartDate
	}
	if req.EndDate != nil {
		updates["end_date"] = *req.EndDate
	}

	if err := storage.DB.WithContext(ctx).Model(project).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("更新项目失败: %w", err)
	}
	return s.GetProjectDetail(ctx, projectID, userID)
}

// DeleteProject 软删除项目并删除项目成员关系。
func (s *ProjectService) DeleteProject(ctx context.Context, projectID, userID uint) error {
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return err
	}
	if project.OwnerID != userID {
		return fmt.Errorf("只有项目 owner 可以删除项目")
	}

	err = storage.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("project_id = ?", projectID).Delete(&model.ProjectMember{}).Error; err != nil {
			return err
		}
		return tx.Delete(project).Error
	})
	if err != nil {
		return fmt.Errorf("删除项目失败: %w", err)
	}
	return nil
}

// AddMember 添加项目成员，owner 可添加 admin/member，admin 只能添加 member。
func (s *ProjectService) AddMember(ctx context.Context, projectID, operatorID uint, req *request.AddProjectMemberRequest) error {
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return err
	}
	operator, err := s.getProjectMember(ctx, projectID, operatorID)
	if err != nil {
		return err
	}
	if !model.HasAdminPermission(operator.Role) {
		return fmt.Errorf("只有项目 owner 或 admin 可以添加成员")
	}
	if operator.Role == model.RoleAdmin && req.Role != model.RoleMember {
		return fmt.Errorf("admin 只能添加普通成员")
	}
	if _, err := s.getTeamMember(ctx, project.TeamID, req.UserID); err != nil {
		return fmt.Errorf("只能添加该团队的成员")
	}

	var count int64
	if err := storage.DB.WithContext(ctx).Model(&model.ProjectMember{}).
		Where("project_id = ? AND user_id = ?", projectID, req.UserID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("查询项目成员失败: %w", err)
	}
	if count > 0 {
		return fmt.Errorf("该用户已经是项目成员")
	}

	if err := storage.DB.WithContext(ctx).Create(&model.ProjectMember{
		ProjectID: projectID,
		UserID:    req.UserID,
		Role:      req.Role,
	}).Error; err != nil {
		return fmt.Errorf("添加项目成员失败: %w", err)
	}
	return nil
}

// RemoveMember 移除项目成员，admin 不能操作 owner 或其他 admin。
func (s *ProjectService) RemoveMember(ctx context.Context, projectID, operatorID, targetUserID uint) error {
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return err
	}
	operator, err := s.getProjectMember(ctx, projectID, operatorID)
	if err != nil {
		return err
	}
	if !model.HasAdminPermission(operator.Role) {
		return fmt.Errorf("只有项目 owner 或 admin 可以移除成员")
	}
	if project.OwnerID == targetUserID {
		return fmt.Errorf("不能移除项目 owner")
	}

	target, err := s.getProjectMember(ctx, projectID, targetUserID)
	if err != nil {
		return err
	}
	if operator.Role == model.RoleAdmin && target.Role != model.RoleMember {
		return fmt.Errorf("admin 只能移除普通成员")
	}

	result := storage.DB.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, targetUserID).
		Delete(&model.ProjectMember{})
	if result.Error != nil {
		return fmt.Errorf("移除项目成员失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("该用户不是项目成员")
	}
	return nil
}

// LeaveProject 退出项目，owner 不允许直接退出。
func (s *ProjectService) LeaveProject(ctx context.Context, projectID, userID uint) error {
	project, err := s.getProject(ctx, projectID)
	if err != nil {
		return err
	}
	if project.OwnerID == userID {
		return fmt.Errorf("owner 不能直接退出项目，请先转让所有权或删除项目")
	}
	if _, err := s.getProjectMember(ctx, projectID, userID); err != nil {
		return err
	}

	result := storage.DB.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		Delete(&model.ProjectMember{})
	if result.Error != nil {
		return fmt.Errorf("退出项目失败: %w", result.Error)
	}
	return nil
}

// GetProjectStats 获取项目成员数和任务状态统计。
func (s *ProjectService) GetProjectStats(ctx context.Context, projectID, userID uint) (*ProjectStats, error) {
	if _, err := s.getProjectMember(ctx, projectID, userID); err != nil {
		return nil, err
	}

	stats := &ProjectStats{}
	if err := storage.DB.WithContext(ctx).Model(&model.ProjectMember{}).
		Where("project_id = ?", projectID).
		Count(&stats.MemberCount).Error; err != nil {
		return nil, fmt.Errorf("统计项目成员失败: %w", err)
	}
	if err := storage.DB.WithContext(ctx).Model(&model.Task{}).
		Where("project_id = ?", projectID).
		Count(&stats.TaskCount).Error; err != nil {
		return nil, fmt.Errorf("统计项目任务失败: %w", err)
	}

	type statusCount struct {
		Status string
		Count  int64
	}
	var counts []statusCount
	if err := storage.DB.WithContext(ctx).Model(&model.Task{}).
		Select("status, COUNT(*) AS count").
		Where("project_id = ?", projectID).
		Group("status").
		Scan(&counts).Error; err != nil {
		return nil, fmt.Errorf("统计任务状态失败: %w", err)
	}
	for _, item := range counts {
		switch item.Status {
		case "todo":
			stats.TodoCount = item.Count
		case "doing":
			stats.DoingCount = item.Count
		case "done":
			stats.DoneCount = item.Count
		}
	}
	return stats, nil
}

func (s *ProjectService) getProject(ctx context.Context, projectID uint) (*model.Project, error) {
	var project model.Project
	err := storage.DB.WithContext(ctx).First(&project, projectID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("项目不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("查询项目失败: %w", err)
	}
	return &project, nil
}

func (s *ProjectService) getProjectMember(ctx context.Context, projectID, userID uint) (*model.ProjectMember, error) {
	if _, err := s.getProject(ctx, projectID); err != nil {
		return nil, err
	}

	var member model.ProjectMember
	err := storage.DB.WithContext(ctx).
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&member).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("你不是该项目成员")
	}
	if err != nil {
		return nil, fmt.Errorf("查询项目成员失败: %w", err)
	}
	return &member, nil
}

func (s *ProjectService) getTeamMember(ctx context.Context, teamID, userID uint) (*model.TeamMember, error) {
	var team model.Team
	err := storage.DB.WithContext(ctx).First(&team, teamID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, fmt.Errorf("团队不存在")
	}
	if err != nil {
		return nil, fmt.Errorf("查询团队失败: %w", err)
	}

	var member model.TeamMember
	err = storage.DB.WithContext(ctx).
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

func validateProjectDates(startDate, endDate *time.Time) error {
	if startDate != nil && endDate != nil && endDate.Before(*startDate) {
		return fmt.Errorf("结束日期不能早于开始日期")
	}
	return nil
}
