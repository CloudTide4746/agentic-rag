package handler

import (
	"context"
	"teamflow/internal/team"
)

type CreateTeamReq struct {
	ActorID     uint   `json:"-"`
	Name        string `json:"name"`
	Description string `json:"description"`
}
type CreateTeamResp struct {
	Team team.Team `json:"team"`
}
type GetTeamReq struct {
	ActorID uint `json:"-"`
	TeamID  uint `json:"-"`
}
type GetTeamResp struct {
	Team team.Team `json:"team"`
}
type ListTeamsReq struct {
	ActorID  uint `json:"-"`
	Page     int  `json:"page"`
	PageSize int  `json:"page_size"`
}
type ListTeamsResp struct {
	Items    []team.Team `json:"items"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
type UpdateTeamReq struct {
	ActorID     uint    `json:"-"`
	TeamID      uint    `json:"-"`
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Avatar      *string `json:"avatar"`
}
type UpdateTeamResp struct {
	Team team.Team `json:"team"`
}
type DeleteTeamReq struct {
	ActorID uint `json:"-"`
	TeamID  uint `json:"-"`
}
type DeleteTeamResp struct{}

type TeamHandler interface {
	CreateTeam(context.Context, *CreateTeamReq) (*CreateTeamResp, error)
	ListTeams(context.Context, *ListTeamsReq) (*ListTeamsResp, error)
	GetTeam(context.Context, *GetTeamReq) (*GetTeamResp, error)
	UpdateTeam(context.Context, *UpdateTeamReq) (*UpdateTeamResp, error)
	DeleteTeam(context.Context, *DeleteTeamReq) (*DeleteTeamResp, error)
}
