package data

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"reflect"
	"strings"
)

type LazyLoadable interface {
	NewInstance()
	SetLoadFunc(entity string, fn func() (any, error))
	HasLoadFunc(entity string) bool
	DeleteLoadFunc(entity string)
	Load(name string, entity any) (any, error)
	Entities() []string
}

type LazyLoader struct {
	loaderMap map[string]func() (any, error)
}

func (l *LazyLoader) NewInstance() {
	l.loaderMap = make(map[string]func() (any, error))
}

func (l *LazyLoader) SetLoadFunc(entity string, fn func() (any, error)) {
	l.loaderMap[entity] = fn
}

func (l *LazyLoader) DeleteLoadFunc(entity string) {
	delete(l.loaderMap, entity)
}

func (l *LazyLoader) Entities() []string {
	entities := make([]string, 0, len(l.loaderMap))
	for k, _ := range l.loaderMap {
		entities = append(entities, k)
	}
	return entities
}

func (l *LazyLoader) HasLoadFunc(entity string) bool {
	_, ok := l.loaderMap[entity]
	return ok
}

func (l *LazyLoader) Load(name string, entity any) (any, error) {
	typeOf := reflect.TypeOf(entity)
	if fn, ok := l.loaderMap[name]; ok {
		loaded, err := fn()
		logrus.Debugf("LazyLoader.Load: LazyLoader[%p][%+v] loaded[%p][%+v]", l, l, loaded, loaded)
		delete(l.loaderMap, name)
		return loaded, err
	} else {
		return nil, fmt.Errorf("lazy load function for %s[%s] is not set", name, typeOf)
	}
}

func LazyLoadNow[T any](name string, lazyLoader LazyLoadable) (T, error) {
	var entity T
	var err error
	var loaded any
	loaded, err = lazyLoader.Load(name, entity)
	if err != nil {
		return entity, err
	}
	valueOfParent := reflect.ValueOf(lazyLoader)
	valueOfLoaded := reflect.ValueOf(loaded)

	child := reflect.Indirect(valueOfParent).FieldByName(name)
	if child.Type().Kind() == reflect.Pointer {
		child.Set(valueOfLoaded)
		return loaded.(T), err
	} else {
		child.Set(reflect.Indirect(valueOfLoaded))
		return reflect.Indirect(reflect.ValueOf(loaded)).Interface().(T), err
	}
}

type GormRepository[T any, ID comparable] struct {
	db *gorm.DB
}

func NewGormRepository[T any, ID comparable](db *gorm.DB) *GormRepository[T, ID] {
	return &GormRepository[T, ID]{db: db}
}

func (u *GormRepository[T, ID]) FindBy(ctx context.Context, belongTo any) ([]T, error) {
	var entity T
	var entities []T
	associations := findAssociations[T](entity)

	db := u.db.Model(&entity)
	for _, v := range associations {
		if v.FetchMode == FetchEagerMode {
			db = db.Preload(v.Name)
		}
	}

	belongToTable := reflect.TypeOf(belongTo).Name()
	var foreignKey string
	var foreignKeyValue any
	var zero bool
	if foreignKeyValue, zero = findID[any, any](belongTo); zero {
		panic(fmt.Sprintf("FindBy: %s's ID field is empty", belongToTable))
	} else {
		foreignKey = fmt.Sprintf("%s_id", strings.ToLower(belongToTable))
	}
	if err := db.Find(&entities, fmt.Sprintf("%s = ?", foreignKey), foreignKeyValue).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return entities, NotFoundError
		} else {
			return entities, err
		}
	}

	for i := 0; i < len(entities); i++ {
		ent := &entities[i]
		associations = findAssociations[T](*ent)

		switch anyEntity := any(ent).(type) {
		case LazyLoadable:
			anyEntity.NewInstance()
			for _, v := range associations {
				if v.FetchMode == FetchLazyMode {
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfBelongTo(ctx, v.Value, v.ID))
				}
			}
		}
	}
	return entities, nil
}

func (u *GormRepository[T, ID]) findOne(ctx context.Context, entity any, id any) (any, error) {
	associations := findAssociations(entity)

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

	associations = findAssociations(entity)

	switch anyEntity := any(&entity).(type) {
	case LazyLoadable:
		anyEntity.NewInstance()
		for _, v := range associations {
			switch v.FetchMode {
			case FetchLazyMode:
				switch v.Type {
				case BelongTo:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc belong-to entity [%p], association [%p], association_id [%v]", anyEntity, v.Value, v.ID)
					if v.ID != nil {
						anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfBelongTo(ctx, v.Value, v.ID))
					}
				case HasOne:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc has-one entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.Value, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfHasOne(ctx, v.Value, v.ForeignKey, id))
				case HasMany:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc has-many entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.Value, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfHasMany(ctx, v.Value, v.ForeignKey, id))
				case ManyToMany:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc many-to-many entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.Value, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfManyMany(ctx, entity, v.Value, v.Name, v.ForeignKey, id))
				}
			}
		}
	}
	return entity, nil
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
			switch v.FetchMode {
			case FetchLazyMode:
				switch v.Type {
				case BelongTo:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc belong-to entity [%p], association [%p], association_id [%v]", anyEntity, v.Value, v.ID)
					if v.ID != nil {
						anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfBelongTo(ctx, v.Value, v.ID))
					}
				case HasOne:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc has-one entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.Value, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfHasOne(ctx, v.Value, v.ForeignKey, id))
				case HasMany:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc has-many entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.Value, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfHasMany(ctx, v.Value, v.ForeignKey, id))
				case ManyToMany:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc many-to-many entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.Value, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfManyMany(ctx, entity, v.Value, v.Name, v.ForeignKey, id))
				}
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
	var id any
	var zero bool
	var db *gorm.DB = u.db.Session(&gorm.Session{FullSaveAssociations: true})

	if id, zero = findID[T, ID](entity); zero {
		panic("entity.ID is missing")
	}
	update := entity
	switch lazyLoader := any(&entity).(type) {
	case LazyLoadable:
		associations := findAssociations[T](entity)
		for _, association := range associations {
			switch association.FetchMode {
			case FetchLazyMode:
				switch association.Type {
				case BelongTo:
					db = db.Omit(association.Name)
				case HasOne, HasMany, ManyToMany:
					if !reflect.ValueOf(entity).FieldByName(association.Name).IsZero() {
						ass := u.db.Unscoped().Model(&entity).Association(association.Name)
						if ass.Error != nil {
							panic(ass.Error)
						}
						if err := ass.Unscoped().Clear(); err != nil {
							panic(err)
						}
					} else if !lazyLoader.HasLoadFunc(association.Name) {
						ass := u.db.Unscoped().Model(&entity).Association(association.Name)
						if ass.Error != nil {
							panic(ass.Error)
						}
						if err := ass.Unscoped().Clear(); err != nil {
							panic(err)
						}
					} else {
						db = db.Omit(association.Name)
					}
				}
			}
		}
	default:
		u.clearAssociations(entity)
	}

	if err := db.Updates(&update).Error; err != nil {
		return entity, err
	}

	associations := findAssociations[T](update)

	switch anyEntity := any(&update).(type) {
	case LazyLoadable:
		anyEntity.NewInstance()
		for _, v := range associations {
			fieldValue := reflect.Indirect(reflect.ValueOf(anyEntity)).FieldByName(v.Name)
			switch v.FetchMode {
			case FetchLazyMode:
				switch v.Type {
				case BelongTo:
					logrus.Debugf("GormRepository.Update: SetLoadFunc belong-to entity [%p], association [%p], association_id [%v]", anyEntity, v.Value, v.ID)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfBelongTo(ctx, v.Value, v.ID))
					// clear association field value
					if fieldValue.Type().Kind() == reflect.Pointer {
						fieldValue.Set(reflect.ValueOf(v.Value))
					} else {
						fieldValue.Set(reflect.Indirect(reflect.ValueOf(v.Value)))
					}
				case HasOne:
					logrus.Debugf("GormRepository.Update: SetLoadFunc has-one entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.Value, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfHasOne(ctx, v.Value, v.ForeignKey, id))
					// clear association field value
					fieldValue.Set(reflect.Indirect(reflect.ValueOf(v.Value)))
				case HasMany:
					logrus.Debugf("GormRepository.Update: SetLoadFunc has-many entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.Value, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfHasMany(ctx, v.Value, v.ForeignKey, id))
					fieldValue.SetLen(0)
				case ManyToMany:
					logrus.Debugf("GormRepository.Update: SetLoadFunc many-to-many entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.Value, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfManyMany(ctx, entity, v.Value, v.Name, v.ForeignKey, id))
					fieldValue.SetLen(0)
				}
			}
		}
	}

	return update, nil
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

func (u *GormRepository[T, ID]) GetLazyLoadFuncOfBelongTo(ctx context.Context, entity any, id any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfBelongTo: entity [%p] [%+v] id[%v]", entity, entity, id)
	return func() (any, error) {
		idValue := reflect.ValueOf(id)
		if idValue.Type().Kind() == reflect.Pointer && idValue.IsNil() {
			return nil, gorm.ErrRecordNotFound
		}
		if idValue.IsZero() {
			return nil, gorm.ErrRecordNotFound
		}
		if err := u.db.Model(entity).First(entity, "id = ?", id).Error; err != nil {
			return nil, err
		}

		return reflect.ValueOf(entity).Interface(), nil
	}
}

func (u *GormRepository[T, ID]) GetLazyLoadFuncOfHasOne(ctx context.Context, entity any, foreignKey string, foreignKeyValue any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfHasOne: entity [%p] [%+v] foreignKey[%s:%v]", entity, entity, foreignKey, foreignKeyValue)
	return func() (any, error) {
		if err := u.db.Model(entity).First(entity, fmt.Sprintf("%s = ?", foreignKey), foreignKeyValue).Error; err != nil {
			return nil, err
		}
		return reflect.ValueOf(entity).Interface(), nil
	}
}

func (u *GormRepository[T, ID]) GetLazyLoadFuncOfHasMany(ctx context.Context, entity any, foreignKey string, foreignKeyValue any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfHasMany: entity [%p] [%+v] foreignKey[%s:%v]", entity, entity, foreignKey, foreignKeyValue)
	return func() (any, error) {
		if err := u.db.Model(entity).Find(entity, fmt.Sprintf("%s = ?", foreignKey), foreignKeyValue).Error; err != nil {
			return nil, err
		}

		// todo recursive set LazyLoadFunc to entity elements
		//entityValue := reflect.ValueOf(entity)
		//for i := 0; i< entityValue.Len(); i++ {
		//	elementValue:= entityValue.Index(i)
		//	elementIDValue := elementValue.FieldByName("ID")
		//	id := elementIDValue.Interface()
		//	u.findOne(ctx, elementValue.Interface(), id)
		//}
		return reflect.ValueOf(entity).Interface(), nil
	}
}

func (u *GormRepository[T, ID]) GetLazyLoadFuncOfManyMany(ctx context.Context, parent any, entity any, associationName string, foreignKey string, foreignKeyValue any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfManyMany: entity [%p] [%+v] foreignKey[%s:%v]", entity, entity, foreignKey, foreignKeyValue)
	return func() (any, error) {
		if err := u.db.Model(parent).Association(associationName).Find(entity, fmt.Sprintf("%s = ?", foreignKey), foreignKeyValue); err != nil {
			return nil, err
		}
		return reflect.ValueOf(entity).Interface(), nil
	}
}

type GormBelongToRepository[T any, S any, ID comparable] struct {
	*GormRepository[T, ID]
}

func NewGormBelongToRepository[T any, S any, ID comparable](gormRepository *GormRepository[T, ID]) *GormBelongToRepository[T, S, ID] {
	return &GormBelongToRepository[T, S, ID]{GormRepository: gormRepository}
}

func (u *GormBelongToRepository[T, S, ID]) FindBy(ctx context.Context, belongTo S) ([]T, error) {
	return u.GormRepository.FindBy(ctx, belongTo)
}
