package service

import (
	"context"
	"errors"
	"testing"

	"teamflow/internal/faults"
	"teamflow/internal/project"
	"teamflow/internal/project/handler"
)

type fakeProjectRepo struct {
	item    *project.Project
	created bool
	member  bool
}

func (f *fakeProjectRepo) Create(_ context.Context, item project.Project) (*project.Project, error) {
	f.created = true
	return &item, nil
}
func (f *fakeProjectRepo) ListByTeam(context.Context, uint, int, int) ([]project.Project, int64, error) {
	return nil, 0, nil
}
func (f *fakeProjectRepo) Get(context.Context, uint) (*project.Project, error) {
	if f.item == nil {
		return nil, faults.ErrNotFound
	}
	copy := *f.item
	return &copy, nil
}
func (f *fakeProjectRepo) IsMember(context.Context, uint, uint) (bool, error) { return f.member, nil }
func (f *fakeProjectRepo) Update(context.Context, uint, map[string]any) (*project.Project, error) {
	return f.item, nil
}
func (f *fakeProjectRepo) Delete(context.Context, uint) error { return nil }

type fakeTeamAccess struct{ member bool }

func (f fakeTeamAccess) IsMember(context.Context, uint, uint) (bool, error) { return f.member, nil }

func TestCreateProjectRequiresTeamMembership(t *testing.T) {
	repo := &fakeProjectRepo{}
	svc := NewProjectService(repo, fakeTeamAccess{})
	_, err := svc.CreateProject(context.Background(), &handler.CreateProjectReq{ActorID: 7, TeamID: 2, Name: "Project"})
	if !errors.Is(err, faults.ErrForbidden) {
		t.Fatalf("error=%v, want forbidden", err)
	}
	if repo.created {
		t.Fatal("project created without team membership")
	}
}
func TestProjectMembershipRequiresActiveTeamMembership(t *testing.T) {
	repo := &fakeProjectRepo{item: &project.Project{ID: "3", TeamID: "2"}, member: true}
	svc := NewProjectService(repo, fakeTeamAccess{})
	member, err := svc.IsMember(context.Background(), 3, 7)
	if err != nil {
		t.Fatal(err)
	}
	if member {
		t.Fatal("project membership remained active after team membership ended")
	}
}
