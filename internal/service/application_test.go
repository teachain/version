package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
	"github.com/teachain/version/internal/repository"
)

type fakeAppRepo struct {
	items     map[uint]*model.Application
	byName    map[string]uint
	nextID    uint
	saved     *model.Application
	getErr    error
	updateErr error
	deleteErr error
	listErr   error
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
func (f *fakeAppRepo) Update(_ context.Context, a *model.Application) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.items[a.ID] = a
	return nil
}
func (f *fakeAppRepo) Delete(_ context.Context, id uint) error {
	if f.deleteErr != nil {
		return f.deleteErr
	}
	if _, ok := f.items[id]; !ok {
		return repository.ErrNotFound
	}
	delete(f.items, id)
	return nil
}
func (f *fakeAppRepo) Get(_ context.Context, id uint) (*model.Application, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if a, ok := f.items[id]; ok {
		return a, nil
	}
	return nil, repository.ErrNotFound
}
func (f *fakeAppRepo) GetByName(_ context.Context, name string) (*model.Application, error) {
	id, ok := f.byName[name]
	if !ok {
		return nil, nil
	}
	return f.items[id], nil
}
func (f *fakeAppRepo) List(_ context.Context, page, size int) ([]model.Application, int64, error) {
	if f.listErr != nil {
		return nil, 0, f.listErr
	}
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

func TestApplicationService_Update_NotFound(t *testing.T) {
	svc := NewApplicationService(newFakeAppRepo())
	newName := "new"
	_, err := svc.Update(context.Background(), 999, &newName, nil, nil)
	if err == nil {
		t.Fatal("expected not-found")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.NotFound {
		t.Fatalf("kind=%v want NotFound", got.Kind)
	}
}

func TestApplicationService_Update_NoFields(t *testing.T) {
	repo := newFakeAppRepo()
	repo.items[1] = &model.Application{ID: 1, Name: "demo", RepoURL: "https://github.com/o/r", Enabled: true}
	svc := NewApplicationService(repo)
	a, err := svc.Update(context.Background(), 1, nil, nil, nil)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if a.Name != "demo" || !a.Enabled {
		t.Fatalf("app mutated: %+v", a)
	}
}

func TestApplicationService_Update_EmptyName(t *testing.T) {
	repo := newFakeAppRepo()
	repo.items[1] = &model.Application{ID: 1, Name: "demo", RepoURL: "https://github.com/o/r", Enabled: true}
	svc := NewApplicationService(repo)
	empty := ""
	_, err := svc.Update(context.Background(), 1, &empty, nil, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.BadRequest {
		t.Fatalf("kind=%v want BadRequest", got.Kind)
	}
}

func TestApplicationService_Update_InvalidURL(t *testing.T) {
	repo := newFakeAppRepo()
	repo.items[1] = &model.Application{ID: 1, Name: "demo", RepoURL: "https://github.com/o/r", Enabled: true}
	svc := NewApplicationService(repo)
	bad := "https://gitlab.com/o/r"
	_, err := svc.Update(context.Background(), 1, nil, &bad, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.BadRequest {
		t.Fatalf("kind=%v want BadRequest", got.Kind)
	}
}

func TestApplicationService_Update_AllFields(t *testing.T) {
	repo := newFakeAppRepo()
	repo.items[1] = &model.Application{ID: 1, Name: "old", RepoURL: "https://github.com/o/old", Enabled: true}
	svc := NewApplicationService(repo)
	newName := "new"
	newURL := "https://github.com/o/new"
	enabled := false
	a, err := svc.Update(context.Background(), 1, &newName, &newURL, &enabled)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if a.Name != "new" || a.RepoURL != newURL || a.Enabled {
		t.Fatalf("bad app: %+v", a)
	}
}

func TestApplicationService_Update_OnlyEnabled(t *testing.T) {
	repo := newFakeAppRepo()
	repo.items[1] = &model.Application{ID: 1, Name: "demo", RepoURL: "https://github.com/o/r", Enabled: true}
	svc := NewApplicationService(repo)
	enabled := false
	a, err := svc.Update(context.Background(), 1, nil, nil, &enabled)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if a.Enabled {
		t.Fatalf("enabled not changed")
	}
	if a.Name != "demo" || a.RepoURL != "https://github.com/o/r" {
		t.Fatalf("other fields mutated: %+v", a)
	}
}

func TestApplicationService_Update_RepoError(t *testing.T) {
	repo := newFakeAppRepo()
	repo.items[1] = &model.Application{ID: 1, Name: "demo", RepoURL: "https://github.com/o/r", Enabled: true}
	dbErr := errors.New("db down")
	repo.updateErr = dbErr
	svc := NewApplicationService(repo)
	_, err := svc.Update(context.Background(), 1, nil, nil, nil)
	if !errors.Is(err, dbErr) {
		t.Fatalf("err=%v want %v", err, dbErr)
	}
}

func TestApplicationService_Update_GetError(t *testing.T) {
	repo := newFakeAppRepo()
	dbErr := errors.New("db down")
	repo.getErr = dbErr
	svc := NewApplicationService(repo)
	_, err := svc.Update(context.Background(), 1, nil, nil, nil)
	if !errors.Is(err, dbErr) {
		t.Fatalf("err=%v want %v", err, dbErr)
	}
}

func TestApplicationService_Delete_OK(t *testing.T) {
	repo := newFakeAppRepo()
	repo.items[1] = &model.Application{ID: 1, Name: "demo"}
	svc := NewApplicationService(repo)
	if err := svc.Delete(context.Background(), 1); err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if _, ok := repo.items[1]; ok {
		t.Fatal("not deleted")
	}
}

func TestApplicationService_Delete_NotFound(t *testing.T) {
	svc := NewApplicationService(newFakeAppRepo())
	err := svc.Delete(context.Background(), 999)
	if err == nil {
		t.Fatal("expected not-found")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.NotFound {
		t.Fatalf("kind=%v want NotFound", got.Kind)
	}
}

func TestApplicationService_Delete_RepoError(t *testing.T) {
	repo := newFakeAppRepo()
	dbErr := errors.New("db down")
	repo.deleteErr = dbErr
	svc := NewApplicationService(repo)
	err := svc.Delete(context.Background(), 1)
	if !errors.Is(err, dbErr) {
		t.Fatalf("err=%v want %v", err, dbErr)
	}
}

func TestApplicationService_Get_OK(t *testing.T) {
	repo := newFakeAppRepo()
	repo.items[1] = &model.Application{ID: 1, Name: "demo"}
	svc := NewApplicationService(repo)
	a, err := svc.Get(context.Background(), 1)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if a == nil || a.Name != "demo" {
		t.Fatalf("bad app: %+v", a)
	}
}

func TestApplicationService_Get_NotFound(t *testing.T) {
	svc := NewApplicationService(newFakeAppRepo())
	a, err := svc.Get(context.Background(), 999)
	if a != nil {
		t.Fatalf("expected nil app, got %+v", a)
	}
	if err == nil {
		t.Fatal("expected not-found")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.NotFound {
		t.Fatalf("kind=%v want NotFound", got.Kind)
	}
}

func TestApplicationService_Get_RepoError(t *testing.T) {
	repo := newFakeAppRepo()
	dbErr := errors.New("db down")
	repo.getErr = dbErr
	svc := NewApplicationService(repo)
	a, err := svc.Get(context.Background(), 1)
	if a != nil {
		t.Fatalf("expected nil app, got %+v", a)
	}
	if !errors.Is(err, dbErr) {
		t.Fatalf("err=%v want %v", err, dbErr)
	}
}

func TestApplicationService_List_OK(t *testing.T) {
	repo := newFakeAppRepo()
	repo.items[1] = &model.Application{ID: 1, Name: "a"}
	repo.items[2] = &model.Application{ID: 2, Name: "b"}
	svc := NewApplicationService(repo)
	items, total, err := svc.List(context.Background(), 1, 10)
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if total != 2 || len(items) != 2 {
		t.Fatalf("total=%d items=%d", total, len(items))
	}
}

func TestApplicationService_List_InvalidPage(t *testing.T) {
	svc := NewApplicationService(newFakeAppRepo())
	_, _, err := svc.List(context.Background(), 0, 10)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.BadRequest {
		t.Fatalf("kind=%v want BadRequest", got.Kind)
	}
}

func TestApplicationService_List_InvalidSize(t *testing.T) {
	svc := NewApplicationService(newFakeAppRepo())
	_, _, err := svc.List(context.Background(), 1, 0)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.(*apperror.Error); got.Kind != apperror.BadRequest {
		t.Fatalf("kind=%v want BadRequest", got.Kind)
	}
}

func TestApplicationService_List_RepoError(t *testing.T) {
	repo := newFakeAppRepo()
	dbErr := errors.New("db down")
	repo.listErr = dbErr
	svc := NewApplicationService(repo)
	_, _, err := svc.List(context.Background(), 1, 10)
	if !errors.Is(err, dbErr) {
		t.Fatalf("err=%v want %v", err, dbErr)
	}
}