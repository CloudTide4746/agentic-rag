package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"teamflow/internal/faults"
	"teamflow/internal/team"
	"teamflow/internal/team/handler"
)

type Repository interface {
	Create(context.Context, team.Team) (*team.Team, error)
	ListForUser(context.Context, uint, int, int) ([]team.Team, int64, error)
	Get(context.Context, uint) (*team.Team, error)
	IsMember(context.Context, uint, uint) (bool, error)
	Update(context.Context, uint, map[string]any) (*team.Team, error)
	Delete(context.Context, uint) error
}

type TeamService struct{ repo Repository }

var _ handler.TeamHandler = (*TeamService)(nil)

func NewTeamService(repo Repository) *TeamService { return &TeamService{repo: repo} }

func (s *TeamService) CreateTeam(ctx context.Context, req *handler.CreateTeamReq) (*handler.CreateTeamResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	name := strings.TrimSpace(req.Name)
	if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 100 || utf8.RuneCountInString(req.Description) > 500 {
		return nil, fmt.Errorf("invalid team fields: %w", faults.ErrValidation)
	}
	created, err := s.repo.Create(ctx, team.Team{Name: name, Description: req.Description, OwnerID: strconv.FormatUint(uint64(req.ActorID), 10)})
	if err != nil {
		return nil, err
	}
	return &handler.CreateTeamResp{Team: *created}, nil
}
func (s *TeamService) ListTeams(ctx context.Context, req *handler.ListTeamsReq) (*handler.ListTeamsResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	page, size := pageArgs(req.Page, req.PageSize)
	items, total, err := s.repo.ListForUser(ctx, req.ActorID, page, size)
	if err != nil {
		return nil, err
	}
	return &handler.ListTeamsResp{Items: items, Total: total, Page: page, PageSize: size}, nil
}
func (s *TeamService) GetTeam(ctx context.Context, req *handler.GetTeamReq) (*handler.GetTeamResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	item, err := s.requireMember(ctx, req.TeamID, req.ActorID)
	if err != nil {
		return nil, err
	}
	return &handler.GetTeamResp{Team: *item}, nil
}
func (s *TeamService) UpdateTeam(ctx context.Context, req *handler.UpdateTeamReq) (*handler.UpdateTeamResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	item, err := s.repo.Get(ctx, req.TeamID)
	if err != nil {
		return nil, err
	}
	if item.OwnerID != strconv.FormatUint(uint64(req.ActorID), 10) {
		return nil, faults.ErrForbidden
	}
	updates := map[string]any{}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if utf8.RuneCountInString(name) < 2 || utf8.RuneCountInString(name) > 100 {
			return nil, fmt.Errorf("invalid name: %w", faults.ErrValidation)
		}
		updates["name"] = name
	}
	if req.Description != nil {
		if utf8.RuneCountInString(*req.Description) > 500 {
			return nil, faults.ErrValidation
		}
		updates["description"] = *req.Description
	}
	if req.Avatar != nil {
		if utf8.RuneCountInString(*req.Avatar) > 500 {
			return nil, faults.ErrValidation
		}
		updates["avatar"] = *req.Avatar
	}
	if len(updates) == 0 {
		return nil, fmt.Errorf("at least one field is required: %w", faults.ErrValidation)
	}
	updated, err := s.repo.Update(ctx, req.TeamID, updates)
	if err != nil {
		return nil, err
	}
	return &handler.UpdateTeamResp{Team: *updated}, nil
}
func (s *TeamService) DeleteTeam(ctx context.Context, req *handler.DeleteTeamReq) (*handler.DeleteTeamResp, error) {
	if req == nil || req.ActorID == 0 {
		return nil, faults.ErrUnauthenticated
	}
	item, err := s.repo.Get(ctx, req.TeamID)
	if err != nil {
		return nil, err
	}
	if item.OwnerID != strconv.FormatUint(uint64(req.ActorID), 10) {
		return nil, faults.ErrForbidden
	}
	if err := s.repo.Delete(ctx, req.TeamID); err != nil {
		return nil, err
	}
	return &handler.DeleteTeamResp{}, nil
}

func (s *TeamService) IsMember(ctx context.Context, teamID, userID uint) (bool, error) {
	return s.repo.IsMember(ctx, teamID, userID)
}
func (s *TeamService) requireMember(ctx context.Context, teamID, userID uint) (*team.Team, error) {
	item, err := s.repo.Get(ctx, teamID)
	if err != nil {
		return nil, err
	}
	member, err := s.repo.IsMember(ctx, teamID, userID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, faults.ErrForbidden
	}
	return item, nil
}
func pageArgs(page, size int) (int, int) {
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
