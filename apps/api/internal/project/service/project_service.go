package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"teamflow/internal/faults"
	"teamflow/internal/project"
	"teamflow/internal/project/handler"
)

type Repository interface {
	Create(context.Context, project.Project) (*project.Project, error)
	ListByTeam(context.Context, uint, int, int) ([]project.Project, int64, error)
	Get(context.Context, uint) (*project.Project, error)
	IsMember(context.Context, uint, uint) (bool, error)
	Update(context.Context, uint, map[string]any) (*project.Project, error)
	Delete(context.Context, uint) error
}
type TeamAccess interface {
	IsMember(context.Context, uint, uint) (bool, error)
}
type ProjectService struct {
	repo  Repository
	teams TeamAccess
}

var _ handler.ProjectHandler = (*ProjectService)(nil)

func NewProjectService(repo Repository, teams TeamAccess) *ProjectService {
	return &ProjectService{repo: repo, teams: teams}
}

func (s *ProjectService) CreateProject(ctx context.Context, req *handler.CreateProjectReq) (*handler.CreateProjectResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	name := strings.TrimSpace(req.Name)
	if req.TeamID == 0 || utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 100 || utf8.RuneCountInString(req.Description) > 500 {
		return nil, fmt.Errorf("invalid project fields: %w", faults.ErrValidation)
	}
	if req.StartDate != nil && req.EndDate != nil && req.StartDate.After(*req.EndDate) {
		return nil, fmt.Errorf("start_date must not follow end_date: %w", faults.ErrValidation)
	}
	member, err := s.teams.IsMember(ctx, req.TeamID, req.ActorID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, faults.ErrForbidden
	}
	created, err := s.repo.Create(ctx, project.Project{Name: name, Description: req.Description, TeamID: fmt.Sprint(req.TeamID), OwnerID: fmt.Sprint(req.ActorID), Status: "active", StartDate: req.StartDate, EndDate: req.EndDate})
	if err != nil {
		return nil, err
	}
	return &handler.CreateProjectResp{Project: *created}, nil
}
func (s *ProjectService) ListProjects(ctx context.Context, req *handler.ListProjectsReq) (*handler.ListProjectsResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	member, err := s.teams.IsMember(ctx, req.TeamID, req.ActorID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, faults.ErrForbidden
	}
	page, size := normalizePage(req.Page, req.PageSize)
	items, total, err := s.repo.ListByTeam(ctx, req.TeamID, page, size)
	if err != nil {
		return nil, err
	}
	return &handler.ListProjectsResp{Items: items, Total: total, Page: page, PageSize: size}, nil
}
func (s *ProjectService) GetProject(ctx context.Context, req *handler.GetProjectReq) (*handler.GetProjectResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	item, err := s.requireMember(ctx, req.ProjectID, req.ActorID)
	if err != nil {
		return nil, err
	}
	return &handler.GetProjectResp{Project: *item}, nil
}

func (s *ProjectService) UpdateProject(ctx context.Context, req *handler.UpdateProjectReq) (*handler.UpdateProjectResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	item, err := s.requireMember(ctx, req.ProjectID, req.ActorID)
	if err != nil {
		return nil, err
	}
	if item.OwnerID != fmt.Sprint(req.ActorID) {
		return nil, faults.ErrForbidden
	}
	updates := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 100 {
			return nil, faults.ErrValidation
		}
		updates["name"] = name
	}
	if req.Description != nil {
		if utf8.RuneCountInString(*req.Description) > 500 {
			return nil, faults.ErrValidation
		}
		updates["description"] = *req.Description
	}
	if req.Status != nil {
		if *req.Status != "active" && *req.Status != "archived" {
			return nil, faults.ErrValidation
		}
		updates["status"] = *req.Status
	}
	if req.StartDate != nil {
		updates["start_date"] = req.StartDate
	}
	if req.EndDate != nil {
		updates["end_date"] = req.EndDate
	}
	start, end := item.StartDate, item.EndDate
	if req.StartDate != nil {
		start = req.StartDate
	}
	if req.EndDate != nil {
		end = req.EndDate
	}
	if start != nil && end != nil && start.After(*end) {
		return nil, fmt.Errorf("start_date must not follow end_date: %w", faults.ErrValidation)
	}
	if len(updates) == 0 {
		return nil, faults.ErrValidation
	}
	updated, err := s.repo.Update(ctx, req.ProjectID, updates)
	if err != nil {
		return nil, err
	}
	return &handler.UpdateProjectResp{Project: *updated}, nil
}

func (s *ProjectService) DeleteProject(ctx context.Context, req *handler.DeleteProjectReq) (*handler.DeleteProjectResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	item, err := s.requireMember(ctx, req.ProjectID, req.ActorID)
	if err != nil {
		return nil, err
	}
	if item.OwnerID != fmt.Sprint(req.ActorID) {
		return nil, faults.ErrForbidden
	}
	if err = s.repo.Delete(ctx, req.ProjectID); err != nil {
		return nil, err
	}
	return &handler.DeleteProjectResp{}, nil
}
func (s *ProjectService) IsMember(ctx context.Context, projectID, userID uint) (bool, error) {
	item, err := s.repo.Get(ctx, projectID)
	if errors.Is(err, faults.ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	teamID, err := strconv.ParseUint(item.TeamID, 10, 64)
	if err != nil {
		return false, fmt.Errorf("invalid project team id: %w", faults.ErrConflict)
	}
	teamMember, err := s.teams.IsMember(ctx, uint(teamID), userID)
	if err != nil || !teamMember {
		return false, err
	}
	return s.repo.IsMember(ctx, projectID, userID)
}
func (s *ProjectService) requireMember(ctx context.Context, projectID, userID uint) (*project.Project, error) {
	item, err := s.repo.Get(ctx, projectID)
	if err != nil {
		return nil, err
	}
	teamID, parseErr := strconv.ParseUint(item.TeamID, 10, 64)
	if parseErr != nil {
		return nil, faults.ErrConflict
	}
	teamMember, err := s.teams.IsMember(ctx, uint(teamID), userID)
	if err != nil {
		return nil, err
	}
	if !teamMember {
		return nil, faults.ErrForbidden
	}
	ok, err := s.repo.IsMember(ctx, projectID, userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, faults.ErrForbidden
	}
	return item, nil
}
func normalizePage(page, size int) (int, int) {
	if page < 1 {
		page = 1
	}
	if page > 100000 {
		page = 100000
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return page, size
}
