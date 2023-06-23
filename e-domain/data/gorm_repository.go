package data

import (
	"context"
	"errors"
	"fmt"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"reflect"
)

type GormRepository[T any, ID comparable] struct {
	transactionManager TransactionManager
}

func NewGormRepository[T any, ID comparable](transactionManager TransactionManager) *GormRepository[T, ID] {
	return &GormRepository[T, ID]{transactionManager: transactionManager}
}

func (u *GormRepository[T, ID]) getGormDB(ctx context.Context) *gorm.DB {
	db, ok := u.transactionManager.Get(ctx).(*gorm.DB)
	if !ok {
		panic("GormRepository.findOne: fail to get *gorm.DB")
	}
	return db
}

// findOne returns entity
func (u *GormRepository[T, ID]) findOne(ctx context.Context, ptrToEntity any, id any) (any, error) {
	db := u.getGormDB(ctx)
	db = u.preload(db, ptrToEntity)
	if err := db.First(ptrToEntity, "id = ?", id).Error; err != nil {
		if logrus.IsLevelEnabled(logrus.DebugLevel) {
			var idd any
			if id != nil {
				idd = reflect.Indirect(reflect.ValueOf(id))
			}
			entityName := reflect.TypeOf(ptrToEntity).Elem().Name()
			logrus.Debugf("GormRepository.findOne: fail to find id[%v] in %s", idd, entityName)
		}
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
	db := u.getGormDB(ctx)
	db = u.preload(db, ptrToEntity)
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
	db := u.getGormDB(ctx)
	ptrToElement := ptrToEmptyElementOfPtrToSlice(ptrToSlice)
	db = u.preload(db, ptrToElement)

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

func (u *GormRepository[T, ID]) findWithChildTable(ctx context.Context, ptrToSlice any, associationName string, foreignKey string, foreignKeyValue any) (any, error) {
	db := u.getGormDB(ctx)
	ptrToElement := ptrToEmptyElementOfPtrToSlice(ptrToSlice)
	joinQuery, whereQuery := buildQueryForFindWithChildTable(ptrToElement, associationName, foreignKey)

	// select * from users left join credit_cards on users.id = credit_cards.user_id where credit_cards.id = 1
	if err := db.Model(ptrToSlice).Joins(joinQuery).Where(whereQuery, foreignKeyValue).Find(ptrToSlice).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFoundError
		} else {
			return nil, err
		}
	}

	// for each element, set lazy loader
	elementValues := reflect.ValueOf(ptrToSlice).Elem()
	for i := 0; i < elementValues.Len(); i++ {
		value := elementValues.Index(i)
		id := value.FieldByName("ID").Interface()
		u.setLazyLoader(ctx, value.Addr().Interface(), id)
	}

	return reflect.Indirect(reflect.ValueOf(ptrToSlice)).Interface(), nil
}

func buildQueryForFindWithChildTable(ptrToElement any, associationName string, foreignKey string) (string, string) {
	elementType := reflect.TypeOf(ptrToElement).Elem().Name()
	elementTable := fmt.Sprintf("%ss", toSnakeCase(elementType))
	childTable := toSnakeCase(associationName)
	joinQuery := fmt.Sprintf("left join %s on %s.id = %s.%s", childTable, elementTable, childTable, foreignKey)
	whereQuery := fmt.Sprintf("%s.id = ?", childTable)
	return joinQuery, whereQuery
}

func (u *GormRepository[T, ID]) findWithJoinTable(ctx context.Context, ptrToSlice any, associationName string, foreignKey string, foreignKeyValue any) (any, error) {
	db := u.getGormDB(ctx)
	ptrToElement := ptrToEmptyElementOfPtrToSlice(ptrToSlice)
	joinQuery, whereQuery := buildQueryForFindWithJoinTable(ptrToElement, associationName, foreignKey)

	// select * from users left join user_languages on users.id = user_languages.user_id where user_languages.language_id = 1
	if err := db.Model(ptrToSlice).Joins(joinQuery).Where(whereQuery, foreignKeyValue).Find(ptrToSlice).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFoundError
		} else {
			return nil, err
		}
	}

	// for each element, set lazy loader
	elementValues := reflect.ValueOf(ptrToSlice).Elem()
	for i := 0; i < elementValues.Len(); i++ {
		value := elementValues.Index(i)
		id := value.FieldByName("ID").Interface()
		u.setLazyLoader(ctx, value.Addr().Interface(), id)
	}

	return reflect.Indirect(reflect.ValueOf(ptrToSlice)).Interface(), nil
}

func buildQueryForFindWithJoinTable(ptrToElement any, associationName string, foreignKey string) (string, string) {
	elementType := reflect.TypeOf(ptrToElement).Elem().Name()
	elementTable := toSnakeCase(elementType + "s")
	joinTable := fmt.Sprintf("%s_%s", toSnakeCase(elementType), toSnakeCase(associationName))
	elementForeignKey := fmt.Sprintf("%s_id", toSnakeCase(elementType))
	joinQuery := fmt.Sprintf("left join %s on %s.id = %s.%s", joinTable, elementTable, joinTable, elementForeignKey)
	whereQuery := fmt.Sprintf("%s.%s = ?", joinTable, foreignKey)
	return joinQuery, whereQuery
}

func (u *GormRepository[T, ID]) findAssociationsByForeignKey(ctx context.Context, ptrToParent any, ptrToChildren any, associationName string, foreignKey string, foreignKeyValue any) (any, error) {
	db := u.getGormDB(ctx)
	association := db.Model(ptrToParent).Association(associationName)

	if err := association.Find(ptrToChildren, fmt.Sprintf("%s = ?", foreignKey), foreignKeyValue); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, NotFoundError
		} else {
			return nil, err
		}
	}

	// for each element, set lazy loader
	elementValues := reflect.ValueOf(ptrToChildren).Elem()
	for i := 0; i < elementValues.Len(); i++ {
		u.setLazyLoader(ctx, elementValues.Index(i).Addr().Interface(), foreignKeyValue)
	}

	return reflect.Indirect(reflect.ValueOf(ptrToChildren)).Interface(), nil
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
					anyEntity.SetLoadFunc(v.Name, u.GetLazyLoadFuncOfBelongTo(ctx, v.PtrToEntity, v.ID))
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

func (u *GormRepository[T, ID]) FindBy(ctx context.Context, name string, byEntity any) ([]T, error) {
	var entity T
	var entities []T

	byEntityName := name
	byAssName := byEntityName + "s"
	associations := findAssociations(entity)
	foreignKeyValue, zero := findID[any, any](byEntity)
	if zero {
		panic(fmt.Sprintf("FindBy: %s's ID field is empty", byEntityName))
	}

	for _, ass := range associations {
		if ass.Name == byEntityName || ass.Name == byAssName {
			switch ass.Type {
			case BelongTo:
				foreignKey := fmt.Sprintf("%s_id", toSnakeCase(byEntityName))
				if found, err := u.findByForeignKey(ctx, &entities, foreignKey, foreignKeyValue); err != nil {
					return entities, err
				} else {
					return found.([]T), nil
				}
			case HasOne, HasMany:
				if found, err := u.findWithChildTable(ctx, &entities, byAssName, ass.ForeignKey, foreignKeyValue); err != nil {
					return entities, err
				} else {
					return found.([]T), nil
				}
			case ManyToMany:
				foreignKey := fmt.Sprintf("%s_id", toSnakeCase(byEntityName))
				if found, err := u.findWithJoinTable(ctx, &entities, ass.Name, foreignKey, foreignKeyValue); err != nil {
					return entities, err
				} else {
					return found.([]T), nil
				}
			}
		}
	}
	return nil, fmt.Errorf("%T has no association with %T", entity, byEntity)
}

func (u *GormRepository[T, ID]) Create(ctx context.Context, entity T) (T, error) {
	db := u.getGormDB(ctx)
	var created T
	if err := db.Create(&entity).Error; err != nil {
		return created, err
	}

	var id any
	var zero bool
	if id, zero = findID[T, ID](entity); zero {
		panic("entity.ID is missing")
	}
	created = entity
	u.setLazyLoader(ctx, &created, id)
	return created, nil
}

func (u *GormRepository[T, ID]) Update(ctx context.Context, entity T) (T, error) {
	db := u.getGormDB(ctx)
	var id any
	var zero bool

	if id, zero = findID[T, ID](entity); zero {
		panic("entity.ID is missing")
	}

	updateTx := db.Model(&entity).Select("*").Omit("id")
	update := entity
	associations := findAssociations(&entity)

	lazyLoader, _ := any(&entity).(LazyLoadable)

	for _, association := range associations {
		updateTx = updateTx.Omit(association.Name)

		switch association.Type {
		case BelongTo:
		case HasOne, HasMany, ManyToMany:
			associationValue := reflect.ValueOf(entity).FieldByName(association.Name)
			ass := db.Unscoped().Model(&entity).Association(association.Name)
			if ass.Error != nil {
				panic(ass.Error)
			}

			if lazyLoader == nil || !lazyLoader.HasLoadFunc(association.Name) {
				// already loaded or eager association
				u.replaceAssociation(associationValue, ass, association)
			} else {
				// lazy association and not loaded. if zero, no updates.
				if !associationValue.IsZero() {
					// association value should be
					u.replaceAssociation(associationValue, ass, association)
				}
			}
		}
	}

	if err := updateTx.Updates(&update).Error; err != nil {
		return entity, err
	}

	var updated T
	if found, err := u.findOne(ctx, &updated, id); err != nil {
		return updated, err
	} else {
		return found.(T), err
	}
}

func (u *GormRepository[T, ID]) replaceAssociation(associationValue reflect.Value, ass *gorm.Association, association Association) {
	if associationValue.IsZero() {
		if err := ass.Unscoped().Clear(); err != nil {
			panic(err)
		}
	} else {
		if association.Type == HasOne {
			value := associationValue.Interface()
			if err := ass.Unscoped().Replace(&value); err != nil {
				panic(err)
			}
		} else {
			if err := ass.Unscoped().Replace(associationValue.Interface()); err != nil {
				panic(err)
			}
		}
	}
}

func (u *GormRepository[T, ID]) clearAssociations(ctx context.Context, entity T) T {
	db := u.getGormDB(ctx)
	var updated T

	associations := findAssociations(&entity)
	updated = entity

	for _, ass := range associations {
		association := db.Unscoped().Model(&updated).Association(ass.Name)
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
	db := u.getGormDB(ctx)
	if _, zero := findID[T, ID](entity); zero {
		panic("entity.ID is missing")
	}
	u.clearAssociations(ctx, entity)
	if err := db.Delete(&entity).Error; err != nil {
		return err
	}
	return nil
}

// GetLazyLoadFuncOfBelongTo returns entity returning function
func (u *GormRepository[T, ID]) GetLazyLoadFuncOfBelongTo(ctx context.Context, ptrToEntity any, id any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfBelongTo: entity [%p] [%+v] id[%v]", ptrToEntity, ptrToEntity, id)
	return func() (any, error) {
		if id == nil {
			return nil, nil
		}
		idValue := reflect.ValueOf(id)
		if idValue.IsZero() {
			return nil, nil
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
func (u *GormRepository[T, ID]) GetLazyLoadFuncOfManyMany(ctx context.Context, ptrToParent any, ptrToChildren any, associationName string, foreignKey string, foreignKeyValue any) func() (any, error) {
	logrus.Debugf("GormRepository.GetLazyLoadFuncOfManyMany: entity [%p] [%+v] foreignKey[%s:%v]", ptrToChildren, ptrToChildren, foreignKey, foreignKeyValue)
	return func() (any, error) {
		var err error
		if ptrToChildren, err = u.findAssociationsByForeignKey(ctx, ptrToParent, ptrToChildren, associationName, foreignKey, foreignKeyValue); err != nil {
			return nil, err
		}
		return ptrToChildren, nil
	}
}

type GormFindByRepository[T any, S any, ID comparable] struct {
	*GormRepository[T, ID]
}

func NewGormFindByRepository[T any, S any, ID comparable](gormRepository *GormRepository[T, ID]) *GormFindByRepository[T, S, ID] {
	return &GormFindByRepository[T, S, ID]{GormRepository: gormRepository}
}

func (u *GormFindByRepository[T, S, ID]) FindBy(ctx context.Context, name string, byEntity S) ([]T, error) {
	return u.GormRepository.FindBy(ctx, name, byEntity)
}
