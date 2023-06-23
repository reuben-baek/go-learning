package data_test

import (
	"context"
	"errors"
	"fmt"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"testing"
	"time"
)

func TestGormTransactionManager(t *testing.T) {
	type User struct {
		gorm.Model
		Name string
	}

	db := getGormDB()
	db.AutoMigrate(&User{})

	transactionManager := data.NewGormTransactionManager(db)
	userRepository := data.NewGormRepository[User, uint](transactionManager)

	t.Run("commit", func(t *testing.T) {
		ctx := context.Background()
		var created User
		err := transactionManager.Do(ctx, func(ctx context.Context) error {
			var err error
			reuben := User{
				Name: "reuben.b",
			}
			created, err = userRepository.Create(ctx, reuben)
			return err
		})
		assert.Nil(t, err)
		assert.NotEmpty(t, created.ID)

		found, err := userRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Equal(t, created.ID, found.ID)
		assert.Equal(t, created.Name, found.Name)
	})

	t.Run("rollback", func(t *testing.T) {
		ctx := context.Background()
		var created User
		err := transactionManager.Do(ctx, func(ctx context.Context) error {
			reuben := User{
				Name: "reuben.b",
			}
			created, _ = userRepository.Create(ctx, reuben)
			return errors.New("fail to save")
		})
		assert.NotNil(t, err)
		assert.NotEmpty(t, created.ID)

		found, err := userRepository.FindOne(ctx, created.ID)
		assert.Equal(t, data.NotFoundError, err)
		assert.Empty(t, found)
	})
	t.Run("ctx cancel", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancelFunction := context.WithCancel(context.Background())

		var managerDoError error
		go func() {
			managerDoError = transactionManager.Do(ctx, func(ctx context.Context) error {
				time.Sleep(20 * time.Millisecond)
				reuben := User{
					Name: "reuben.b",
				}

				_, err := userRepository.Create(ctx, reuben)
				fmt.Printf("err : %v\n", err)
				return err
			})
		}()

		time.Sleep(10 * time.Millisecond)
		cancelFunction()
		time.Sleep(25 * time.Millisecond)

		assert.NotNil(t, managerDoError)
		assert.Equal(t, errors.New("context canceled"), managerDoError)
	})
	t.Run("rollback on panic", func(t *testing.T) {
		ctx := context.Background()
		var created User
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("panic: %v", r)
				}
			}()
			transactionManager.Do(ctx, func(ctx context.Context) error {
				var err error
				reuben := User{
					Name: "reuben.b",
				}
				created, err = userRepository.Create(ctx, reuben)
				panic("something wrong")
				return err
			})
		}()
		found, err := userRepository.FindOne(ctx, created.ID)
		assert.Equal(t, data.NotFoundError, err)
		assert.Empty(t, found)
	})
}
