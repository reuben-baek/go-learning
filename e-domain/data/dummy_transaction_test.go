package data_test

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDummyTransactionManager(t *testing.T) {
	transactionManager := data.NewDummyTransactionManager()

	type User struct {
		ID   string
		Name string
	}
	repository := data.NewInMemoryRepository[User, string](transactionManager)

	ctx := context.Background()
	reuben := User{
		ID:   "reuben.b",
		Name: "reuben baek",
	}

	err := transactionManager.Do(ctx, func(ctx context.Context) error {
		_, err := repository.Create(ctx, reuben)
		return err
	})
	assert.Nil(t, err)
}
