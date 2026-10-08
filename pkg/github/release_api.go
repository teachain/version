package github

import (
	"context"
	"fmt"

	goGithub "github.com/google/go-github/v60/github"
)

type apiRepo struct {
	client *goGithub.Client
}

func NewClient(token string) ReleaseRepository {
	var client *goGithub.Client
	if token != "" {
		ts := (&goGithub.BasicAuthTransport{Username: "x-access-token", Password: token}).Client()
		client = goGithub.NewClient(ts)
	} else {
		client = goGithub.NewClient(nil)
	}
	return &apiRepo{client: client}
}

func (r *apiRepo) FetchReleases(ctx context.Context, repoURL string) ([]Release, error) {
	owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, fmt.Errorf("parse repo url: %w", err)
	}
	opt := &goGithub.ListOptions{PerPage: 100}
	var all []Release
	for page := 1; ; page++ {
		opt.Page = page
		rs, resp, err := r.client.Repositories.ListReleases(ctx, owner, repo, opt)
		if err != nil {
			return nil, fmt.Errorf("list releases: %w", err)
		}
		for _, gh := range rs {
			all = append(all, Release{
				TagName:     gh.GetTagName(),
				Name:        gh.GetName(),
				Body:        gh.GetBody(),
				URL:         gh.GetHTMLURL(),
				PublishedAt: gh.GetPublishedAt().Time,
			})
		}
		if resp.NextPage == 0 {
			break
		}
	}
	return all, nil
}
