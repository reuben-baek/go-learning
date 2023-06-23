package data

import (
	"context"
	"github.com/sirupsen/logrus"
)

type InMemoryRepository[T any, ID comparable] struct {
	database           map[ID]T
	transactionManager TransactionManager
}

func NewInMemoryRepository[T any, ID comparable](transactionManager TransactionManager) *InMemoryRepository[T, ID] {
	return &InMemoryRepository[T, ID]{
		database:           make(map[ID]T),
		transactionManager: transactionManager,
	}
}

func (u *InMemoryRepository[T, ID]) FindBy(ctx context.Context, name string, byEntity any) ([]T, error) {
	//TODO implement me
	panic("implement me")
}

func (u *InMemoryRepository[T, ID]) FindOne(ctx context.Context, id ID) (T, error) {
	var v T
	var ok bool
	if v, ok = u.database[id]; ok {
		return v, nil
	} else {
		return v, NotFoundError
	}
}

func (u *InMemoryRepository[T, ID]) Create(ctx context.Context, entity T) (T, error) {
	transaction := u.transactionManager.Get(ctx)
	logrus.Infof("InMemoryRepository.Create: transaction [%v] entity [%+v]", transaction, entity)
	id, _ := findID[T, ID](entity)
	u.database[id] = entity
	return entity, nil
}

func (u *InMemoryRepository[T, ID]) Update(ctx context.Context, entity T) (T, error) {
	var v T
	id, _ := findID[T, ID](entity)
	if _, ok := u.database[id]; ok {
		u.database[id] = entity
		return entity, nil
	} else {
		return v, NotFoundError
	}
}

func (u *InMemoryRepository[T, ID]) Delete(ctx context.Context, entity T) error {
	id, _ := findID[T, ID](entity)
	if _, ok := u.database[id]; ok {
		delete(u.database, id)
		return nil
	} else {
		return NotFoundError
	}
}
