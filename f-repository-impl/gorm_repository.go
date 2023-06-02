package f_repository_impl

import (
	"context"
	"errors"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type GormRepository[T any, ID comparable] struct {
	db *gorm.DB
}

func NewGormRepository[T any, ID comparable](db *gorm.DB) *GormRepository[T, ID] {
	return &GormRepository[T, ID]{db: db}
}

func (u *GormRepository[T, ID]) FindOne(ctx context.Context, id ID) (T, error) {
	var entity T
	preloads := findPreloadModels[T](entity)

	db := u.db.Model(&entity)
	for _, v := range preloads {
		db = db.Preload(v)
	}
	if err := db.First(&entity, "id = ?", id).Error; err != nil {
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
	updated := u.clearAssociations(entity)
	if err := u.db.Save(&entity).Error; err != nil {
		return updated, err
	}
	updated = entity
	return updated, nil
}

func (u *GormRepository[T, ID]) clearAssociations(entity T) T {
	var updated T

	associations := findAssociations[T](entity)
	updated = entity

	for _, ass := range associations {
		association := u.db.Unscoped().Model(&updated).Association(ass.Name)
		if association.Error != nil {
			panic(association.Error)
		}
		logrus.Debugf("GormRepository.Update: Association %s %s", association.Relationship.Type, ass.Name)
		switch association.Relationship.Type {
		case schema.BelongsTo:
		case schema.HasOne:
			if err := association.Unscoped().Clear(); err != nil {
				panic(err)
			}
		case schema.HasMany:
			if err := association.Unscoped().Clear(); err != nil {
				panic(err)
			}
		case schema.Many2Many:
			if err := association.Clear(); err != nil {
				panic(err)
			}
		}
	}
	return updated
}

func (u *GormRepository[T, ID]) Delete(ctx context.Context, entity T) error {
	if _, zero := findID[T, ID](entity); zero {
		panic("entity.ID is missing")
	}
	u.clearAssociations(entity)
	if err := u.db.Delete(&entity).Error; err != nil {
		return err
	}
	return nil
}
