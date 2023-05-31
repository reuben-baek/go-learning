package f_repository_impl_test

import (
	"context"
	f_repository_impl "github.com/reuben-baek/go-learning/f-repository-impl"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"testing"
	"time"
)

func NewGormUserRepository(db *gorm.DB) UserRepository {
	return f_repository_impl.NewGormRepository[User, string](db)
}

func TestGormUserRepository(t *testing.T) {
	logConfig := logger.New(log.New(os.Stdout, "\r\n", log.LstdFlags), logger.Config{
		SlowThreshold: 100 * time.Millisecond,
		LogLevel:      logger.Info,
		Colorful:      true,
	})
	var err error
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{Logger: logConfig})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&User{})

	var userRepository UserRepository
	userRepository = NewGormUserRepository(db)

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
