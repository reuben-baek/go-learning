package data

import (
	"context"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type GormTransactionManager struct {
	db *gorm.DB
}

func NewGormTransactionManager(db *gorm.DB) *GormTransactionManager {
	return &GormTransactionManager{db: db}
}

type gormTransactionKey struct{}

func (g *GormTransactionManager) Do(ctx context.Context, f func(ctx context.Context) error) error {
	tx := g.db.Begin().WithContext(ctx)
	newCtx := context.WithValue(ctx, gormTransactionKey{}, tx)

	panicked := true
	defer func() {
		if panicked {
			tx.Rollback()
		}
	}()

	err := f(newCtx)
	panicked = false // if f is panicked, this statement is not executed.

	if err != nil {
		tx.Rollback()
		return err
	}
	tx.Commit()
	return nil
}

func (g *GormTransactionManager) Get(ctx context.Context) any {
	tx, ok := ctx.Value(gormTransactionKey{}).(*gorm.DB)
	if !ok {
		logrus.Warnf("GormTransactionManager.Get: no transaction session")
		tx = g.db.WithContext(ctx)
	}
	return tx
}
