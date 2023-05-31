package f_repository_impl

import (
	"context"
	"errors"
	"gorm.io/gorm"
)

type GormRepository[T any, ID comparable] struct {
	db *gorm.DB
}

func NewGormRepository[T any, ID comparable](db *gorm.DB) *GormRepository[T, ID] {
	return &GormRepository[T, ID]{db: db}
}

func (u *GormRepository[T, ID]) FindOne(ctx context.Context, id ID) (T, error) {
	var entity T
	if err := u.db.First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity, NotFoundError
		} else {
			return entity, err
		}
	}

	return entity, nil
}

func (u *GormRepository[T, ID]) Create(ctx context.Context, entity T) (T, error) {
	var created T
	if err := u.db.Create(&entity).Error; err != nil {
		return created, err
	}
	created = entity
	return created, nil
}

func (u *GormRepository[T, ID]) Update(ctx context.Context, entity T) (T, error) {
	if _, zero := findID[T, ID](entity); zero {
		panic("entity.ID is missing")
	}

	var updated T
	if err := u.db.Save(&entity).Error; err != nil {
		return updated, err
	}
	updated = entity
	return updated, nil
}

func (u *GormRepository[T, ID]) Delete(ctx context.Context, entity T) error {
	if _, zero := findID[T, ID](entity); zero {
		panic("entity.ID is missing")
	}
	if err := u.db.Delete(&entity).Error; err != nil {
		return err
	}
	return nil
}
