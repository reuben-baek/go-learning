package f_repository_impl_test

import (
	"context"
	e_domain "github.com/reuben-baek/go-learning/e-domain"
	f_repository_impl "github.com/reuben-baek/go-learning/f-repository-impl"
	"github.com/stretchr/testify/assert"
	"testing"
)

type User struct {
	ID   string
	Name string
}

type UserRepository interface {
	e_domain.Repository[User, string]
}

func NewInMemoryUserRepository() UserRepository {
	return f_repository_impl.NewInMemoryRepository[User, string]()
}

func TestUserRepository(t *testing.T) {
	var userRepository UserRepository
	userRepository = NewInMemoryUserRepository()

	ctx := context.Background()
	reuben := User{
		ID: "reuben.b",
	}
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
	assert.ErrorIs(t, f_repository_impl.NotFoundError, err)
}
