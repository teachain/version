package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/teachain/version/internal/model"
)

func TestVersionRepo_CRUD(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	appRepo := NewApplicationXormRepository(eng)
	verRepo := NewVersionXormRepository(eng)
	ctx := context.Background()

	app := &model.Application{Name: "vcrud", RepoURL: "https://github.com/octocat/Hello-World", Enabled: true}
	if err := appRepo.Save(ctx, app); err != nil {
		t.Fatalf("save app: %v", err)
	}

	now := time.Now()
	v1 := &model.Version{ApplicationID: app.ID, TagName: "v1.0.0", Name: "first", PublishedAt: now}
	v2 := &model.Version{ApplicationID: app.ID, TagName: "v1.1.0", Name: "second", PublishedAt: now}
	if err := verRepo.Save(ctx, v1); err != nil {
		t.Fatalf("save v1: %v", err)
	}
	if err := verRepo.Save(ctx, v2); err != nil {
		t.Fatalf("save v2: %v", err)
	}

	if err := verRepo.Save(ctx, &model.Version{ApplicationID: app.ID, TagName: "v1.0.0", PublishedAt: now}); err == nil {
		t.Fatal("expected duplicate unique-index error")
	}

	page, total, err := verRepo.PageByApp(ctx, app.ID, 0, 10)
	if err != nil || len(page) != 2 || total != 2 {
		t.Fatalf("page: %v len=%d total=%d", err, len(page), total)
	}

	tags, err := verRepo.ListTagNamesByApp(ctx, app.ID)
	if err != nil || len(tags) != 2 {
		t.Fatalf("tags: %v %v", err, tags)
	}
}

func TestVersionRepo_Get_NotFound(t *testing.T) {
	eng, cleanup := setupEngine(t)
	defer cleanup()
	verRepo := NewVersionXormRepository(eng)
	ctx := context.Background()

	_, err := verRepo.Get(ctx, 99999)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
