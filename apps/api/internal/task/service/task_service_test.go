package service

import (
	"context"
	"errors"
	"testing"

	"teamflow/internal/faults"
	"teamflow/internal/task"
	"teamflow/internal/task/handler"
)

type fakeTaskRepo struct {
	created bool
	updated bool
	item    *task.Task
}

func (f *fakeTaskRepo) Create(_ context.Context, item task.Task) (*task.Task, error) {
	f.created = true
	return &item, nil
}
func (f *fakeTaskRepo) ListByProject(context.Context, uint, int, int) ([]task.Task, int64, error) {
	return nil, 0, nil
}
func (f *fakeTaskRepo) Get(context.Context, uint) (*task.Task, error) {
	if f.item == nil {
		return nil, faults.ErrNotFound
	}
	copy := *f.item
	return &copy, nil
}
func (f *fakeTaskRepo) Update(context.Context, uint, int, map[string]any) (*task.Task, error) {
	f.updated = true
	return f.item, nil
}
func (f *fakeTaskRepo) Delete(context.Context, uint) error { return nil }

type fakeProjectAccess struct{ member bool }

func (f fakeProjectAccess) IsMember(context.Context, uint, uint) (bool, error) { return f.member, nil }

func TestCreateTaskRequiresProjectMembership(t *testing.T) {
	repo := &fakeTaskRepo{}
	svc := NewTaskService(repo, fakeProjectAccess{})
	_, err := svc.CreateTask(context.Background(), &handler.CreateTaskReq{ActorID: 7, ProjectID: "2", Title: "Task"})
	if !errors.Is(err, faults.ErrForbidden) {
		t.Fatalf("error=%v, want forbidden", err)
	}
	if repo.created {
		t.Fatal("task created without project membership")
	}
}
func TestListTasksRequiresProjectID(t *testing.T) {
	svc := NewTaskService(&fakeTaskRepo{}, fakeProjectAccess{member: true})
	_, err := svc.ListTasks(context.Background(), &handler.ListTasksReq{ActorID: 7})
	if !errors.Is(err, faults.ErrValidation) {
		t.Fatalf("error=%v, want validation error", err)
	}
}
func TestCreateTaskRejectsUnsupportedPriority(t *testing.T) {
	repo := &fakeTaskRepo{}
	svc := NewTaskService(repo, fakeProjectAccess{member: true})
	_, err := svc.CreateTask(context.Background(), &handler.CreateTaskReq{ActorID: 7, ProjectID: "2", Title: "Task", Priority: "urgent"})
	if !errors.Is(err, faults.ErrValidation) {
		t.Fatalf("error=%v, want validation error", err)
	}
	if repo.created {
		t.Fatal("task created with invalid priority")
	}
}

func TestUpdateTaskRejectsStaleVersion(t *testing.T) {
	repo := &fakeTaskRepo{item: &task.Task{ID: "4", ProjectID: "2", Version: 3}}
	svc := NewTaskService(repo, fakeProjectAccess{member: true})
	title := "new title"
	version := 2
	_, err := svc.UpdateTask(context.Background(), &handler.UpdateTaskReq{ActorID: 7, TaskID: 4, Version: &version, Title: &title})
	if !errors.Is(err, faults.ErrConflict) {
		t.Fatalf("error = %v, want conflict", err)
	}
	if repo.updated {
		t.Fatal("stale update reached repository")
	}
}

func TestUpdateTaskRequiresVersion(t *testing.T) {
	repo := &fakeTaskRepo{item: &task.Task{ID: "4", ProjectID: "2", Version: 0}}
	svc := NewTaskService(repo, fakeProjectAccess{member: true})
	title := "new title"
	_, err := svc.UpdateTask(context.Background(), &handler.UpdateTaskReq{ActorID: 7, TaskID: 4, Title: &title})
	if !errors.Is(err, faults.ErrValidation) {
		t.Fatalf("error = %v, want validation", err)
	}
}
