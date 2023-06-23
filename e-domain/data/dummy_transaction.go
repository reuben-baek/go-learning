package data

import (
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type dummyTransactionManager struct {
}

func NewDummyTransactionManager() *dummyTransactionManager {
	return &dummyTransactionManager{}
}

type dummyTransactionKey struct{}

func (d *dummyTransactionManager) Do(ctx context.Context, f func(ctx context.Context) error) error {
	transactionID := uuid.New()
	logrus.Infof("DummyTransactionManager.Do: transaction [%s]", transactionID)
	return f(context.WithValue(ctx, dummyTransactionKey{}, &transactionID))
}

func (d *dummyTransactionManager) Get(ctx context.Context) any {
	tx := ctx.Value(dummyTransactionKey{}).(*uuid.UUID)
	if tx == nil {
		return uuid.New()
	}
	return tx
}
