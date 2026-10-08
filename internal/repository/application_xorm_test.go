package repository

import (
	"context"
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
		t.Fatalf("failation: %v", err)
	}
	if err := eng.Sync(new(model.Application)); err != nil {
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
