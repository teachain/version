package github

import (
	"context"
	"time"
)

type Release struct {
	TagName     string
	Name        string
	Body        string
	URL         string
	PublishedAt time.Time
}

type ReleaseRepository interface {
	FetchReleases(ctx context.Context, repoURL string) ([]Release, error)
}