package service

import (
	"context"
	"testing"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
)

type fakeVersionRepo struct {
	items  map[uint]*model.Version
	nextID uint
}

func newFakeVersionRepo() *fakeVersionRepo {
	return &fakeVersionRepo{items: map[uint]*model.Version{}}
}

func (f *fakeVersionRepo) Save(_ context.Context, v *model.Version) error {
	f.nextID++
	v.ID = f.nextID
	f.items[v.ID] = v
	return nil
}
func (f *fakeVersionRepo) Get(_ context.Context, id uint) (*model.Version, error) {
	if v, ok := f.items[id]; ok {
		return v, nil
	}
	return nil, apperror.NotFoundf("version %d", id)
}
func (f *fakeVersionRepo) PageByApp(_ context.Context, appID uint, offset, limit int) ([]model.Version, int64, error) {
	var out []model.Version
	for _, v := range f.items {
		if v.ApplicationID == appID {
			out = append(out, *v)
		}
	}
	return out, int64(len(out)), nil
}
func (f *fakeVersionRepo) ListTagNamesByApp(_ context.Context, appID uint) ([]string, error) {
	seen := map[string]struct{}{}
	for _, v := range f.items {
		if v.ApplicationID == appID {
			seen[v.TagName] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for tg := range seen {
		out = append(out, tg)
	}
	return out, nil
}

func TestVersionService_ListByApp_InvalidPaging(t *testing.T) {
	_, _, err := NewVersionService(newFakeVersionRepo()).ListByApp(context.Background(), 1, 0, 10)
	if got := err.(*apperror.Error); got.Kind != apperror.BadRequest {
		t.Fatalf("kind=%v", got.Kind)
	}
}

func TestVersionService_ListByApp_OK(t *testing.T) {
	repo := newFakeVersionRepo()
	_ = repo.Save(context.Background(), &model.Version{ApplicationID: 1, TagName: "v1"})
	items, total, err := NewVersionService(repo).ListByApp(context.Background(), 1, 1, 10)
	if err != nil || total != 1 || len(items) != 1 {
		t.Fatalf("err=%v total=%d items=%d", err, total, len(items))
	}
}
