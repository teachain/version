package poller

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/teachain/version/internal/model"
	"github.com/teachain/version/internal/repository"
	gh "github.com/teachain/version/pkg/github"
)

type Poller struct {
	interval    time.Duration
	concurrency int
	apps        repository.ApplicationRepository
	versions    repository.VersionRepository
	releases    gh.ReleaseRepository
	log         *zap.Logger
}

func New(apps repository.ApplicationRepository, versions repository.VersionRepository, releases gh.ReleaseRepository, interval time.Duration, concurrency int, log *zap.Logger) *Poller {
	return &Poller{apps: apps, versions: versions, releases: releases, interval: interval, concurrency: concurrency, log: log}
}

func (p *Poller) Run(ctx context.Context) {
	t := time.NewTicker(p.interval)
	defer t.Stop()
	p.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.tick(ctx)
		}
	}
}

func (p *Poller) tick(ctx context.Context) {
	apps, err := p.apps.ListEnabled(ctx)
	if err != nil {
		p.log.Error("list enabled applications", zap.Error(err))
		return
	}
	sem := make(chan struct{}, p.concurrency)
	var wg sync.WaitGroup
	for _, a := range apps {
		wg.Add(1)
		sem <- struct{}{}
		go func(a model.Application) {
			defer wg.Done()
			defer func() { <-sem }()
			p.process(ctx, a)
		}(a)
	}
	wg.Wait()
}

func (p *Poller) process(ctx context.Context, a model.Application) {
	defer func() {
		if err := p.apps.UpdateLastCheckAt(ctx, a.ID, time.Now()); err != nil {
			p.log.Error("update last_check_at", zap.Uint("appID", a.ID), zap.Error(err))
		}
	}()
	releases, err := p.releases.FetchReleases(ctx, a.RepoURL)
	if err != nil {
		p.log.Error("fetch releases", zap.Uint("appID", a.ID), zap.String("repoURL", a.RepoURL), zap.Error(err))
		return
	}
	tags, err := p.versions.ListTagNamesByApp(ctx, a.ID)
	if err != nil {
		p.log.Error("list tag names", zap.Uint("appID", a.ID), zap.Error(err))
		return
	}
	known := make(map[string]struct{}, len(tags))
	for _, tg := range tags {
		known[tg] = struct{}{}
	}
	for _, r := range releases {
		if _, ok := known[r.TagName]; ok {
			continue
		}
		v := &model.Version{
			ApplicationID: a.ID,
			TagName:       r.TagName,
			Name:          r.Name,
			Body:          r.Body,
			URL:           r.URL,
			PublishedAt:   r.PublishedAt,
		}
		if err := p.versions.Save(ctx, v); err != nil {
			p.log.Error("save version", zap.Uint("appID", a.ID), zap.String("tag", r.TagName), zap.Error(err))
		}
	}
}
