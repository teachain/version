package service

import (
	"context"
	"testing"
	"time"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
)

type fakeAppRepo struct {
	items  map[uint]*model.Application
	byName map[string]uint
	nextID uint
	saved  *model.Application
}

func newFakeAppRepo() *fakeAppRepo {
	return &fakeAppRepo{items: map[uint]*model.Application{}, byName: map[string]uint{}}
}

func (f *fakeAppRepo) Save(_ context.Context, a *model.Application) error {
	f.nextID++
	a.ID = f.nextID
	f.items[a.ID] = a
	f.byName[a.Name] = a.ID
	f.saved = a
	return nil
}
func (f *fakeAppRepo) Update(_ context.Context, a *model.Application) error { f.items[a.ID] = a; return nil }
func (f *fakeAppRepo) Delete(_ context.Context, id uint) error              { delete(f.items, id); return nil }
func (f *fakeAppRepo) Get(_ context.Context, id uint) (*model.Application, error) {
	if a, ok := f.items[id]; ok {
		return a, nil
	}
	return nil, apperror.NotFoundf("app %d", id)
}
func (f *fakeAppRepo) GetByName(_ context.Context, name string) (*model.Application, error) {
	id, ok := f.byName[name]
	if !ok {
		return nil, nil
	}
	return f.items[id], nil
}
func (f *fakeAppRepo) List(_ context.Context, page, size int) ([]model.Application, int64, error) {
	out := make([]model.Application, 0, len(f.items))
	for _, a := range f.items {
		out = append(out, *a)
	}
	return out, int64(len(out)), nil
}
func (f *fakeAppRepo) ListEnabled(_ context.Context) ([]model.Application, error) {
	out := make([]model.Application, 0)
	for _, a := range f.items {
		if a.Enabled {
			out = append(out, *a)
		}
	}
	return out, nil
}
func (f *fakeAppRepo) UpdateLastCheckAt(_ context.Context, id uint, t time.Time) error {
	if a, ok := f.items[id]; ok {
		a.LastCheckAt = &t
		return nil
	}
	return apperror.NotFoundf("app %d", id)
}

func TestApplicationService_Create_OK(t *testing.T) {
	repo := newFakeAppRepo()
	svc := NewApplicationService(repo)
	a, err := svc.Create(context.Background(), "demo", "https://github.com/octocat/Hello-World")
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if a.Name != "demo" || a.ID == 0 {
		t.Fatalf("bad app: %+v", a)
	}
}

func TestApplicationService_Create_InvalidURL(t *testing.T) {
	svc := NewApplicationService(newFakeAppRepo())
	_, err := svc.Create(context.Background(), "x", "https://gitlab.com/o/r")
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.BadRequest {
		t.Fatalf("kind=%v", got.Kind)
	}
}

func TestApplicationService_Create_Conflict(t *testing.T) {
	repo := newFakeAppRepo()
	svc := NewApplicationService(repo)
	_, _ = svc.Create(context.Background(), "demo", "https://github.com/octocat/Hello-World")
	_, err := svc.Create(context.Background(), "demo", "https://github.com/other/repo")
	if err == nil {
		t.Fatal("expected conflict")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.Conflict {
		t.Fatalf("kind=%v", got.Kind)
	}
}