package data_test

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/stretchr/testify/assert"
	"testing"
)

type User struct {
	ID   string
	Name string
}

type UserRepository interface {
	data.Repository[User, string]
}

func NewUserRepository(transactionManager data.TransactionManager) UserRepository {
	return data.NewInMemoryRepository[User, string](transactionManager)
}

func TestUserRepository(t *testing.T) {
	var userRepository UserRepository
	transactionManager := data.NewDummyTransactionManager()
	userRepository = NewUserRepository(transactionManager)

	ctx := context.Background()
	reuben := User{
		ID: "reuben.b",
	}
	transactionManager.Do(ctx, func(ctx context.Context) error {
		created, err := userRepository.Create(ctx, reuben)
		assert.Nil(t, err)
		assert.Equal(t, reuben.ID, created.ID)

		found, err := userRepository.FindOne(ctx, reuben.ID)
		assert.Nil(t, err)
		assert.Equal(t, reuben.ID, found.ID)

		reubenUpdate := reuben
		reubenUpdate.Name = "reuben baek"
		updated, err := userRepository.Update(ctx, reubenUpdate)
		assert.Nil(t, err)
		assert.Equal(t, reubenUpdate.ID, updated.ID)
		assert.Equal(t, reubenUpdate.Name, updated.Name)

		foundUpdated, err := userRepository.FindOne(ctx, reuben.ID)
		assert.Nil(t, err)
		assert.Equal(t, reubenUpdate.ID, foundUpdated.ID)
		assert.Equal(t, reubenUpdate.Name, foundUpdated.Name)

		err = userRepository.Delete(ctx, reuben)
		assert.Nil(t, err)

		_, err = userRepository.FindOne(ctx, reuben.ID)
		assert.ErrorIs(t, data.NotFoundError, err)
		return nil
	})
}
