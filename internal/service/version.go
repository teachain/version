package service

import (
	"context"
	"errors"

	"github.com/teachain/version/internal/apperror"
	"github.com/teachain/version/internal/model"
	"github.com/teachain/version/internal/repository"
)

type VersionService interface {
	ListByApp(ctx context.Context, appID uint, page, size int) ([]model.Version, int64, error)
}

type versionService struct {
	repo repository.VersionRepository
}

func NewVersionService(repo repository.VersionRepository) VersionService {
	return &versionService{repo: repo}
}

func (s *versionService) ListByApp(ctx context.Context, appID uint, page, size int) ([]model.Version, int64, error) {
	if page < 1 || size < 1 {
		return nil, 0, apperror.BadRequestf("invalid pagination")
	}
	offset := (page - 1) * size
	items, total, err := s.repo.PageByApp(ctx, appID, offset, size)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, 0, apperror.NotFoundf("application %d", appID)
		}
		return nil, 0, err
	}
	return items, total, nil
}
