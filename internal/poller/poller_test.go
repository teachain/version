package poller

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/teachain/version/internal/model"
	gh "github.com/teachain/version/pkg/github"
)

type stubApps struct {
	list []model.Application
	updates atomic.Uint64
}

func (s *stubApps) Save(_ context.Context, _ *model.Application) error { return nil }
func (s *stubApps) Update(_ context.Context, _ *model.Application) error { return nil }
func (s *stubApps) Delete(_ context.Context, _ uint) error              { return nil }
func (s *stubApps) Get(_ context.Context, id uint) (*model.Application, error) {
	return nil, nil
}
func (s *stubApps) GetByName(_ context.Context, _ string) (*model.Application, error) {
	return nil, nil
}
func (s *stubApps) List(_ context.Context, _, _ int) ([]model.Application, int64, error) {
	return nil, 0, nil
}
func (s *stubApps) ListEnabled(_ context.Context) ([]model.Application, error) {
	return s.list, nil
}
func (s *stubApps) UpdateLastCheckAt(_ context.Context, _ uint, _ time.Time) error {
	s.updates.Add(1)
	return nil
}

type stubVersions struct{}

func (s *stubVersions) Save(_ context.Context, _ *model.Version) error { return nil }
func (s *stubVersions) Get(_ context.Context, _ uint) (*model.Version, error) {
	return nil, nil
}
func (s *stubVersions) PageByApp(_ context.Context, _ uint, _, _ int) ([]model.Version, int64, error) {
	return nil, 0, nil
}
func (s *stubVersions) ListTagNamesByApp(_ context.Context, _ uint) ([]string, error) {
	return []string{"v1.0.0"}, nil
}

type stubReleases struct{}

func (s *stubReleases) FetchReleases(_ context.Context, _ string) ([]gh.Release, error) {
	return []gh.Release{{TagName: "v1.0.0"}, {TagName: "v1.1.0"}}, nil
}

func zapNop() *zap.Logger { return zap.NewNop() }

func TestPoller_PollsOnce(t *testing.T) {
	apps := &stubApps{list: []model.Application{{ID: 42, Name: "x", RepoURL: "https://github.com/octocat/Hello-World", Enabled: true}}}
	p := New(apps, &stubVersions{}, &stubReleases{}, 10*time.Millisecond, 2, zapNop())
	ctx, cancel := context.WithCancel(context.Background())
	go p.Run(ctx)
	time.Sleep(40 * time.Millisecond)
	cancel()
	if apps.updates.Load() == 0 {
		t.Fatal("expected at least one last_check_at write")
	}
}
