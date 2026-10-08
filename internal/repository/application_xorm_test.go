package repository

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/teachain/version/internal/model"
	"github.com/testcontainers/testcontainers-go"
	tmysql "github.com/testcontainers/testcontainers-go/modules/mysql"
	"xorm.io/xorm"
)

func setupEngine(t *testing.T) (*xorm.Engine, func()) {
	t.Helper()
	if os.Getenv("TEST_DB_DSN") != "" {
		eng, err := NewEngine(os.Getenv("TEST_DB_DSN"))
		if err != nil {
			t.Fatalf("engine: %v", err)
		}
		if err := eng.Sync(new(model.Application)); err != nil {
			t.Fatalf("sync: %v", err)
		}
		if err := eng.Sync(new(model.Version)); err != nil {
			t.Fatalf("sync: %v", err)
		}
		return eng, func() { _ = eng.Close() }
	}
	ctx := context.Background()
	c, err := tmysql.RunContainer(ctx, testcontainers.WithImage("mysql:8.0"))
	if err != nil {
		t.Skipf("no docker: %v", err)
	}
	dsn, err := c.ConnectionString(ctx)
	if err != nil {
		t.Fatalf("dsn: %v", err)
	}
	eng, err := NewEngine(dsn)
	if err != nil {
		t.Fatalf("engine: %v", err)
	}
	if err := eng.Sync(new(model.Application)); err != nil {
		t.Fatalf("sync: %v", err)
	}
	if err := eng.Sync(new(model.Version)); err != nil {
		t.Fatalf("sync: %v", err)
	}
	return eng, func() {
		_ = eng.Close()
		_ = c.Terminate(ctx)
	}
}

func TestApplicationRepo_CRUD(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	repo := NewApplicationXormRepository(eng)
	ctx := context.Background()

	a := &model.Application{Name: "foo", RepoURL: "https://github.com/octocat/Hello-World", Enabled: true}
	if err := repo.Save(ctx, a); err != nil {
		t.Fatalf("save: %v", err)
	}
	if a.ID == 0 {
		t.Fatal("id not set")
	}

	got, err := repo.Get(ctx, a.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "foo" {
		t.Fatalf("name=%q", got.Name)
	}

	got.Name = "bar"
	if err := repo.Update(ctx, got); err != nil {
		t.Fatalf("update: %v", err)
	}

	dup, err := repo.GetByName(ctx, "bar")
	if err != nil || dup == nil {
		t.Fatalf("getbyname: %v %v", dup, err)
	}

	list, total, err := repo.List(ctx, 1, 10)
	if err != nil || total < 1 || len(list) < 1 {
		t.Fatalf("list: %v total=%d", err, total)
	}

	if err := repo.UpdateLastCheckAt(ctx, a.ID, time.Now()); err != nil {
		t.Fatalf("lastcheck: %v", err)
	}

	if err := repo.Delete(ctx, a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestApplicationRepo_ListEnabled(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	repo := NewApplicationXormRepository(eng)
	ctx := context.Background()

	enabled := &model.Application{Name: "list-enabled-on", RepoURL: "https://github.com/octocat/Hello-World", Enabled: true}
	if err := repo.Save(ctx, enabled); err != nil {
		t.Fatalf("save enabled: %v", err)
	}
	disabled := &model.Application{Name: "list-enabled-off", RepoURL: "https://github.com/octocat/Hello-World", Enabled: false}
	if err := repo.Save(ctx, disabled); err != nil {
		t.Fatalf("save disabled: %v", err)
	}

	got, err := repo.ListEnabled(ctx)
	if err != nil {
		t.Fatalf("list enabled: %v", err)
	}
	var names []string
	for _, a := range got {
		names = append(names, a.Name)
	}
	hasEnabled, hasDisabled := false, false
	for _, n := range names {
		switch n {
		case "list-enabled-on":
			hasEnabled = true
		case "list-enabled-off":
			hasDisabled = true
		}
	}
	if !hasEnabled {
		t.Fatalf("expected enabled app in result, got %v", names)
	}
	if hasDisabled {
		t.Fatalf("did not expect disabled app in result, got %v", names)
	}
}

func TestApplicationRepo_UpdateLastCheckAt_NotFound(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	repo := NewApplicationXormRepository(eng)
	ctx := context.Background()

	err := repo.UpdateLastCheckAt(ctx, 99999, time.Now())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestApplicationRepo_Delete_NotFound(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	repo := NewApplicationXormRepository(eng)
	ctx := context.Background()

	err := repo.Delete(ctx, 99999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestApplicationRepo_Get_NotFound(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	repo := NewApplicationXormRepository(eng)
	ctx := context.Background()

	_, err := repo.Get(ctx, 99999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestApplicationRepo_GetByName_Miss(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	repo := NewApplicationXormRepository(eng)
	ctx := context.Background()

	got, err := repo.GetByName(ctx, "does-not-exist")
	if err != nil {
		t.Fatalf("expected nil error for miss, got %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil result for miss, got %+v", got)
	}
}
