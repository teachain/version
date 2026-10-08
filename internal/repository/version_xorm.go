package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/teachain/version/internal/model"
	"xorm.io/xorm"
)

type versionXormRepo struct {
	eng *xorm.Engine
}

func NewVersionXormRepository(eng *xorm.Engine) VersionRepository {
	return &versionXormRepo{eng: eng}
}

func (r *versionXormRepo) Save(ctx context.Context, v *model.Version) error {
	sess := r.eng.Context(ctx)
	affected, err := sess.Insert(v)
	if err != nil {
		return fmt.Errorf("insert version: %w", err)
	}
	if affected == 0 {
		return errors.New("insert version: 0 rows affected")
	}
	return nil
}

func (r *versionXormRepo) Get(ctx context.Context, id uint) (*model.Version, error) {
	sess := r.eng.Context(ctx)
	v := new(model.Version)
	has, err := sess.ID(id).Get(v)
	if err != nil {
		return nil, fmt.Errorf("get version: %w", err)
	}
	if !has {
		return nil, fmt.Errorf("version %d: %w", id, ErrNotFound)
	}
	return v, nil
}

func (r *versionXormRepo) PageByApp(ctx context.Context, appID uint, offset, limit int) ([]model.Version, int64, error) {
	sess := r.eng.Context(ctx)
	total, err := sess.Where("application_id = ?", appID).Count(new(model.Version))
	if err != nil {
		return nil, 0, fmt.Errorf("count versions: %w", err)
	}
	var out []model.Version
	err = sess.Where("application_id = ?", appID).
		OrderBy("published_at DESC, id DESC").
		Limit(limit, offset).
		Find(&out)
	if err != nil {
		return nil, 0, fmt.Errorf("page versions: %w", err)
	}
	return out, total, nil
}

func (r *versionXormRepo) ListTagNamesByApp(ctx context.Context, appID uint) ([]string, error) {
	sess := r.eng.Context(ctx)
	rows, err := sess.Table(new(model.Version)).
		Where("application_id = ?", appID).
		Cols("tag_name").
		Rows(new(model.Version))
	if err != nil {
		return nil, fmt.Errorf("list tag names: %w", err)
	}
	defer rows.Close()
	tags := make([]string, 0)
	for rows.Next() {
		v := new(model.Version)
		if err := rows.Scan(v); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		tags = append(tags, v.TagName)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tag names: %w", err)
	}
	return tags, nil
}
