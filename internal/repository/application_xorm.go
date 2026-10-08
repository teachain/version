package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/teachain/version/internal/model"
	"xorm.io/xorm"
)

type applicationXormRepo struct {
	eng *xorm.Engine
}

func NewApplicationXormRepository(eng *xorm.Engine) ApplicationRepository {
	return &applicationXormRepo{eng: eng}
}

func (r *applicationXormRepo) Save(ctx context.Context, a *model.Application) error {
	sess := r.eng.Context(ctx)
	affected, err := sess.Insert(a)
	if err != nil {
		return fmt.Errorf("insert application: %w", err)
	}
	if affected == 0 {
		return errors.New("insert application: 0 rows affected")
	}
	return nil
}

func (r *applicationXormRepo) Update(ctx context.Context, a *model.Application) error {
	sess := r.eng.Context(ctx)
	affected, err := sess.ID(a.ID).Cols("name", "repo_url", "enabled").Update(a)
	if err != nil {
		return fmt.Errorf("update application: %w", err)
	}
	if affected == 0 {
		return errors.New("update application: 0 rows affected")
	}
	return nil
}

func (r *applicationXormRepo) Delete(ctx context.Context, id uint) error {
	sess := r.eng.Context(ctx)
	affected, err := sess.ID(id).Delete(&model.Application{})
	if err != nil {
		return fmt.Errorf("delete application: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("application %d: %w", id, ErrNotFound)
	}
	return nil
}

func (r *applicationXormRepo) Get(ctx context.Context, id uint) (*model.Application, error) {
	sess := r.eng.Context(ctx)
	a := new(model.Application)
	has, err := sess.ID(id).Get(a)
	if err != nil {
		return nil, fmt.Errorf("get application: %w", err)
	}
	if !has {
		return nil, fmt.Errorf("application %d: %w", id, ErrNotFound)
	}
	return a, nil
}

func (r *applicationXormRepo) GetByName(ctx context.Context, name string) (*model.Application, error) {
	sess := r.eng.Context(ctx)
	a := new(model.Application)
	has, err := sess.Where("name = ?", name).Get(a)
	if err != nil {
		return nil, fmt.Errorf("get application by name: %w", err)
	}
	if !has {
		return nil, nil
	}
	return a, nil
}

func (r *applicationXormRepo) List(ctx context.Context, page, size int) ([]model.Application, int64, error) {
	sess := r.eng.Context(ctx)
	total, err := sess.Count(new(model.Application))
	if err != nil {
		return nil, 0, fmt.Errorf("count applications: %w", err)
	}
	var out []model.Application
	err = sess.Limit(size, (page-1)*size).Find(&out)
	if err != nil {
		return nil, 0, fmt.Errorf("list applications: %w", err)
	}
	return out, total, nil
}

func (r *applicationXormRepo) ListEnabled(ctx context.Context) ([]model.Application, error) {
	sess := r.eng.Context(ctx)
	var out []model.Application
	if err := sess.Where("enabled = ?", true).Find(&out); err != nil {
		return nil, fmt.Errorf("list enabled: %w", err)
	}
	return out, nil
}

func (r *applicationXormRepo) UpdateLastCheckAt(ctx context.Context, id uint, t time.Time) error {
	sess := r.eng.Context(ctx)
	a := &model.Application{LastCheckAt: &t}
	affected, err := sess.ID(id).Cols("last_check_at").Update(a)
	if err != nil {
		return fmt.Errorf("update last_check_at: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("application %d: %w", id, ErrNotFound)
	}
	return nil
}
