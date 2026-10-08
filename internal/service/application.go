package service

import (
	"context"
	"errors"

	gh "github.com/teachain/version/pkg/github"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
	"github.com/teachain/version/internal/repository"
)

type ApplicationService interface {
	Create(ctx context.Context, name, repoURL string) (*model.Application, error)
	Update(ctx context.Context, id uint, name, repoURL *string, enabled *bool) (*model.Application, error)
	Delete(ctx context.Context, id uint) error
	Get(ctx context.Context, id uint) (*model.Application, error)
	List(ctx context.Context, page, size int) ([]model.Application, int64, error)
}

type applicationService struct {
	repo repository.ApplicationRepository
}

func NewApplicationService(repo repository.ApplicationRepository) ApplicationService {
	return &applicationService{repo: repo}
}

func (s *applicationService) Create(ctx context.Context, name, repoURL string) (*model.Application, error) {
	if name == "" {
		return nil, apperror.BadRequestf("name required")
	}
	if _, _, err := gh.ParseRepoURL(repoURL); err != nil {
		return nil, apperror.BadRequestf("invalid repo_url: %v", err)
	}
	existing, err := s.repo.GetByName(ctx, name)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, apperror.Conflictf("application %q already exists", name)
	}
	a := &model.Application{Name: name, RepoURL: repoURL, Enabled: true}
	if err := s.repo.Save(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *applicationService) Update(ctx context.Context, id uint, name, repoURL *string, enabled *bool) (*model.Application, error) {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.NotFoundf("application %d", id)
		}
		return nil, err
	}
	if name != nil {
		if *name == "" {
			return nil, apperror.BadRequestf("name cannot be empty")
		}
		a.Name = *name
	}
	if repoURL != nil {
		if _, _, err := gh.ParseRepoURL(*repoURL); err != nil {
			return nil, apperror.BadRequestf("invalid repo_url: %v", err)
		}
		a.RepoURL = *repoURL
	}
	if enabled != nil {
		a.Enabled = *enabled
	}
	if err := s.repo.Update(ctx, a); err != nil {
		return nil, err
	}
	return a, nil
}

func (s *applicationService) Delete(ctx context.Context, id uint) error {
	if err := s.repo.Delete(ctx, id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.NotFoundf("application %d", id)
		}
		return err
	}
	return nil
}

func (s *applicationService) Get(ctx context.Context, id uint) (*model.Application, error) {
	a, err := s.repo.Get(ctx, id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.NotFoundf("application %d", id)
		}
		return nil, err
	}
	return a, nil
}

func (s *applicationService) List(ctx context.Context, page, size int) ([]model.Application, int64, error) {
	if page < 1 || size < 1 {
		return nil, 0, apperror.BadRequestf("invalid pagination")
	}
	return s.repo.List(ctx, page, size)
}