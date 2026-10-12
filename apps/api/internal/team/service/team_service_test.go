package service

import (
	"context"
	"errors"
	"testing"

	"teamflow/internal/faults"
	"teamflow/internal/team"
	"teamflow/internal/team/handler"
)

type fakeTeamRepo struct {
	item   *team.Team
	member bool
	called string
}

func (f *fakeTeamRepo) Create(_ context.Context, item team.Team) (*team.Team, error) {
	f.called = "create"
	return &item, nil
}
func (f *fakeTeamRepo) ListForUser(_ context.Context, _ uint, page, size int) ([]team.Team, int64, error) {
	f.called = "list"
	return nil, 0, nil
}
func (f *fakeTeamRepo) Get(context.Context, uint) (*team.Team, error) {
	f.called = "get"
	if f.item == nil {
		return nil, faults.ErrNotFound
	}
	copy := *f.item
	return &copy, nil
}
func (f *fakeTeamRepo) IsMember(context.Context, uint, uint) (bool, error) { return f.member, nil }
func (f *fakeTeamRepo) Update(context.Context, uint, map[string]any) (*team.Team, error) {
	f.called = "update"
	return f.item, nil
}
func (f *fakeTeamRepo) Delete(context.Context, uint) error { f.called = "delete"; return nil }

func TestGetTeamRejectsNonMember(t *testing.T) {
	repo := &fakeTeamRepo{item: &team.Team{ID: "3", OwnerID: "7"}}
	svc := NewTeamService(repo)
	_, err := svc.GetTeam(context.Background(), &handler.GetTeamReq{ActorID: 8, TeamID: 3})
	if !errors.Is(err, faults.ErrForbidden) {
		t.Fatalf("error = %v, want forbidden", err)
	}
}
func TestUpdateTeamRequiresOwner(t *testing.T) {
	repo := &fakeTeamRepo{item: &team.Team{ID: "3", OwnerID: "7"}}
	svc := NewTeamService(repo)
	name := "new"
	_, err := svc.UpdateTeam(context.Background(), &handler.UpdateTeamReq{ActorID: 8, TeamID: 3, Name: &name})
	if !errors.Is(err, faults.ErrForbidden) {
		t.Fatalf("error = %v, want forbidden", err)
	}
	if repo.called == "update" {
		t.Fatal("unauthorized update reached repository")
	}
}
func TestListTeamsBoundsPageSize(t *testing.T) {
	repo := &fakeTeamRepo{}
	svc := NewTeamService(repo)
	resp, err := svc.ListTeams(context.Background(), &handler.ListTeamsReq{ActorID: 1, Page: 0, PageSize: 500})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Page != 1 || resp.PageSize != 100 {
		t.Fatalf("page=%d size=%d, want 1 and 100", resp.Page, resp.PageSize)
	}
}
