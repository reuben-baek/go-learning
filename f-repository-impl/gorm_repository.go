package f_repository_impl

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"reflect"
)

type LazyLoadable interface {
	NewInstance()
	SetLazyLoadFunc(entity string, fn func() (any, error))
	LoadNow(entity any) (any, error)
}

type LazyLoader struct {
	loaderMap map[string]func() (any, error)
}

func (l *LazyLoader) NewInstance() {
	l.loaderMap = make(map[string]func() (any, error))
}

func (l *LazyLoader) SetLazyLoadFunc(entity string, fn func() (any, error)) {
	l.loaderMap[entity] = fn
}

func (l *LazyLoader) LoadNow(entity any) (any, error) {
	typeOf := reflect.TypeOf(entity)
	if fn, ok := l.loaderMap[typeOf.Name()]; ok {
		return fn()
	} else {
		panic(fmt.Sprintf("lazy load funtion for %s is not set", typeOf))
	}
}

func LazyLoadNow[T any](lazyLoader LazyLoadable) (T, error) {
	var entity T
	var err error
	var loaded any
	loaded, err = lazyLoader.LoadNow(entity)
	return loaded.(T), err
}

type GormRepository[T any, ID comparable] struct {
	db *gorm.DB
}

func NewGormRepository[T any, ID comparable](db *gorm.DB) *GormRepository[T, ID] {
	return &GormRepository[T, ID]{db: db}
}

func (u *GormRepository[T, ID]) FindOne(ctx context.Context, id ID) (T, error) {
	var entity T

	associations := findAssociations[T](entity)

	db := u.db.Model(&entity)
	for _, v := range associations {
		if v.FetchMode == FetchEagerMode {
			db = db.Preload(v.Name)
		}
	}
	if err := db.First(&entity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entity, NotFoundError
		} else {
			return entity, err
		}
	}

	associations = findAssociations[T](entity)

	switch anyEntity := any(&entity).(type) {
	case LazyLoadable:
		anyEntity.NewInstance()
		for _, v := range associations {
			if v.FetchMode == FetchLazyMode {
				anyEntity.SetLazyLoadFunc(v.Name, u.GetLazyLoadFunc(ctx, v.Value, v.ID))
			}
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

func (u *GormRepository[T, ID]) GetLazyLoadFunc(ctx context.Context, entity any, id any) func() (any, error) {
	logrus.Infof("entity: %+v id: %v", entity, id)
	return func() (any, error) {
		if err := u.db.Model(entity).First(entity, "id = ?", id).Error; err != nil {
			return nil, err
		}
		return reflect.Indirect(reflect.ValueOf(entity)).Interface(), nil
	}
}
