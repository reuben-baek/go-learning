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
	Load(name string, emptyEntity any) (any, error) // Load returns entity
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

func (l *LazyLoader) Load(name string, emptyEntity any) (any, error) {
	typeOf := reflect.TypeOf(emptyEntity)
	if fn, ok := l.loaderMap[name]; ok {
		loadedEntity, err := fn()
		logrus.Debugf("LazyLoader.Load: LazyLoader[%p] loaded [%s] [%+v], err[%v]", l, typeOf.String(), loadedEntity, err)
		delete(l.loaderMap, name)
		return loadedEntity, err
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
		logrus.Debugf("LazyLoadNow: parent[%s] field[%s %s] value[%s %s]", valueOfParent.Type().String(), name, child.Type().String(), valueOfLoaded.Type().String(), valueOfLoaded.Interface())
		child.Set(reflect.New(reflect.TypeOf(loaded)))
		child.Elem().Set(valueOfLoaded)
		return child.Interface().(T), err
	} else {
		logrus.Debugf("LazyLoadNow: parent[%s] field[%s %s] value[%s %s]", valueOfParent.Type().String(), name, child.Type().String(), valueOfLoaded.Type().String(), valueOfLoaded.Interface())
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

// findOne returns entity
func (u *GormRepository[T, ID]) findOne(ctx context.Context, ptrToEntity any, id any) (any, error) {
	db := u.preload(u.db, ptrToEntity)
	if err := db.First(ptrToEntity, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFoundError
		} else {
			return nil, err
		}
	}

	u.setLazyLoader(ctx, ptrToEntity, id)

	return reflect.Indirect(reflect.ValueOf(ptrToEntity)).Interface(), nil
}

// findOneByForeignKey returns ptrEoEntity
func (u *GormRepository[T, ID]) findOneByForeignKey(ctx context.Context, ptrToEntity any, foreignKey string, id any) (any, error) {
	db := u.preload(u.db, ptrToEntity)
	if err := db.Model(ptrToEntity).First(ptrToEntity, fmt.Sprintf("%s = ?", foreignKey), id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFoundError
		} else {
			return nil, err
		}
	}

	u.setLazyLoader(ctx, ptrToEntity, id)
	return reflect.Indirect(reflect.ValueOf(ptrToEntity)).Interface(), nil
}

func (u *GormRepository[T, ID]) findByForeignKey(ctx context.Context, ptrToSlice any, foreignKey string, id any) (any, error) {
	ptrToElement := ptrToEmptyElementOfPtrToSlice(ptrToSlice)
	db := u.preload(u.db, ptrToElement)

	if err := db.Model(ptrToSlice).Find(ptrToSlice, fmt.Sprintf("%s = ?", foreignKey), id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFoundError
		} else {
			return nil, err
		}
	}

	// for each element, set lazy loader
	elementValues := reflect.ValueOf(ptrToSlice).Elem()
	for i := 0; i < elementValues.Len(); i++ {
		u.setLazyLoader(ctx, elementValues.Index(i).Addr().Interface(), id)
	}

	return reflect.Indirect(reflect.ValueOf(ptrToSlice)).Interface(), nil
}

func (u *GormRepository[T, ID]) setLazyLoader(ctx context.Context, ptrToEntity any, id any) any {
	associations := findAssociations(ptrToEntity)

	switch anyEntity := ptrToEntity.(type) {
	case LazyLoadable:
		anyEntity.NewInstance()
		for _, v := range associations {
			switch v.FetchMode {
			case FetchLazyMode:
				switch v.Type {
				case BelongTo:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc belong-to entity [%p], association [%p], association_id [%v]", anyEntity, v.PtrToEntity, v.ID)
					if v.ID != nil {
						anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfBelongTo(ctx, v.PtrToEntity, v.ID))
					}
				case HasOne:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc has-one entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.PtrToEntity, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfHasOne(ctx, v.PtrToEntity, v.ForeignKey, id))
				case HasMany:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc has-many entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.PtrToEntity, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfHasMany(ctx, v.PtrToEntity, v.ForeignKey, id))
				case ManyToMany:
					logrus.Debugf("GormRepository.FindOne: SetLoadFunc many-to-many entity [%p], association [%p], foreignKey [%s], foreignKeyValue [%v]", anyEntity, v.PtrToEntity, v.ForeignKey, id)
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfManyMany(ctx, ptrToEntity, v.PtrToEntity, v.Name, v.ForeignKey, id))
				}
			}
		}
	}
	return ptrToEntity
}

func (u *GormRepository[T, ID]) preload(db *gorm.DB, ptrToEntity any) *gorm.DB {
	associations := findAssociations(ptrToEntity)

	tx := db.Model(ptrToEntity)
	for _, v := range associations {
		if v.FetchMode == FetchEagerMode {
			tx = tx.Preload(v.Name)
		}
	}
	return tx
}

func (u *GormRepository[T, ID]) FindOne(ctx context.Context, id ID) (T, error) {
	var entity T
	if found, err := u.findOne(ctx, &entity, id); err != nil {
		return entity, err
	} else {
		return found.(T), nil
	}
}

func (u *GormRepository[T, ID]) FindBy(ctx context.Context, belongTo any) ([]T, error) {
	var entities []T

	belongToTable := reflect.TypeOf(belongTo).Name()
	if foreignKeyValue, zero := findID[any, any](belongTo); zero {
		panic(fmt.Sprintf("FindBy: %s's ID field is empty", belongToTable))
	} else {
		foreignKey := fmt.Sprintf("%s_id", strings.ToLower(belongToTable))
		if found, err := u.findByForeignKey(ctx, &entities, foreignKey, foreignKeyValue); err != nil {
			return entities, err
		} else {
			return found.([]T), nil
		}
	}
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
		associations := findAssociations(&entity)
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

	var updated T
	if found, err := u.findOne(ctx, &updated, id); err != nil {
		return updated, err
	} else {
		return found.(T), err
	}
}

func (u *GormRepository[T, ID]) clearAssociations(entity T) T {
	var updated T

	associations := findAssociations(&entity)
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

// GetLazyLoadFuncOfBelongTo returns entity returning function
func (u *GormRepository[T, ID]) GetLazyLoadFuncOfBelongTo(ctx context.Context, ptrToEntity any, id any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfBelongTo: entity [%p] [%+v] id[%v]", ptrToEntity, ptrToEntity, id)
	return func() (any, error) {
		idValue := reflect.ValueOf(id)
		if idValue.Type().Kind() == reflect.Pointer && idValue.IsNil() {
			return nil, gorm.ErrRecordNotFound
		}
		if idValue.IsZero() {
			return nil, gorm.ErrRecordNotFound
		}
		if found, err := u.findOne(ctx, ptrToEntity, id); err != nil {
			return nil, err
		} else {
			return found, nil
		}
	}
}

// GetLazyLoadFuncOfHasOne returns entity returning function
func (u *GormRepository[T, ID]) GetLazyLoadFuncOfHasOne(ctx context.Context, ptrToEntity any, foreignKey string, foreignKeyValue any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfHasOne: entity [%p] [%+v] foreignKey[%s:%v]", ptrToEntity, ptrToEntity, foreignKey, foreignKeyValue)
	return func() (any, error) {
		var err error
		if ptrToEntity, err = u.findOneByForeignKey(ctx, ptrToEntity, foreignKey, foreignKeyValue); err != nil {
			return nil, err
		}
		return ptrToEntity, nil
	}
}

// GetLazyLoadFuncOfHasMany returns slice of entity returning function
func (u *GormRepository[T, ID]) GetLazyLoadFuncOfHasMany(ctx context.Context, ptrToEntity any, foreignKey string, foreignKeyValue any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfHasMany: entity [%p] [%+v] foreignKey[%s:%v]", ptrToEntity, ptrToEntity, foreignKey, foreignKeyValue)
	return func() (any, error) {
		var err error
		if ptrToEntity, err = u.findByForeignKey(ctx, ptrToEntity, foreignKey, foreignKeyValue); err != nil {
			return nil, err
		}
		return ptrToEntity, nil
	}
}

// GetLazyLoadFuncOfManyMany returns slice of entity returning function
func (u *GormRepository[T, ID]) GetLazyLoadFuncOfManyMany(ctx context.Context, ptrToParent any, ptrToChild any, associationName string, foreignKey string, foreignKeyValue any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfManyMany: entity [%p] [%+v] foreignKey[%s:%v]", ptrToChild, ptrToChild, foreignKey, foreignKeyValue)
	return func() (any, error) {
		var err error
		if ptrToChild, err = u.findAssociationsByForeignKey(ctx, ptrToParent, ptrToChild, associationName, foreignKey, foreignKeyValue); err != nil {
			return nil, err
		}
		return ptrToChild, nil
	}
}

func (u *GormRepository[T, ID]) findAssociationsByForeignKey(ctx context.Context, ptrToParent any, ptrToChild any, associationName string, foreignKey string, foreignKeyValue any) (any, error) {
	db := u.db.Model(ptrToParent).Association(associationName)

	if err := db.Find(ptrToChild, fmt.Sprintf("%s = ?", foreignKey), foreignKeyValue); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFoundError
		} else {
			return nil, err
		}
	}

	// for each element, set lazy loader
	elementValues := reflect.ValueOf(ptrToChild).Elem()
	for i := 0; i < elementValues.Len(); i++ {
		u.setLazyLoader(ctx, elementValues.Index(i).Addr().Interface(), foreignKeyValue)
	}

	return reflect.Indirect(reflect.ValueOf(ptrToChild)).Interface(), nil
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
