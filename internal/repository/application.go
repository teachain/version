package repository

import (
	"context"
	"time"

	"github.com/teachain/version/internal/model"
)

type ApplicationRepository interface {
	Save(ctx context.Context, a *model.Application) error
	Update(ctx context.Context, a *model.Application) error
	Delete(ctx context.Context, id uint) error
	Get(ctx context.Context, id uint) (*model.Application, error)
	GetByName(ctx context.Context, name string) (*model.Application, error)
	List(ctx context.Context, page, size int) ([]model.Application, int64, error)
	ListEnabled(ctx context.Context) ([]model.Application, error)
	UpdateLastCheckAt(ctx context.Context, id uint, t time.Time) error
}
