package repository

import (
	"context"

	"github.com/teachain/version/internal/model"
)

type VersionRepository interface {
	Save(ctx context.Context, v *model.Version) error
	Get(ctx context.Context, id uint) (*model.Version, error)
	PageByApp(ctx context.Context, appID uint, offset, limit int) ([]model.Version, int64, error)
	ListTagNamesByApp(ctx context.Context, appID uint) ([]string, error)
}
