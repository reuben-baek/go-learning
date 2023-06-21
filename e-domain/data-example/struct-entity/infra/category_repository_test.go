package infra_test

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/infra"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"testing"
	"time"
)

func TestCategoryRepository(t *testing.T) {
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

	db.AutoMigrate(&infra.Category{})

	categoryGormRepository := data.NewGormRepository[infra.Category, uint](db)
	categoryRepository := infra.NewCategoryRepository(data.NewDtoWrapRepository[infra.Category, domain.Category, uint](categoryGormRepository))

	t.Run("create & find", func(t *testing.T) {
		ctx := context.Background()
		deviceCategory := domain.Category{
			Name: "device",
		}
		deviceCategory, err := categoryRepository.Create(ctx, deviceCategory)

		computerCategory := domain.Category{
			Name:   "computer",
			Parent: data.LazyLoadValue[domain.Category](deviceCategory),
		}
		created, err := categoryRepository.Create(ctx, computerCategory)
		assert.Nil(t, err)
		assert.Equal(t, computerCategory.Name, created.Name)
		assert.NotEmpty(t, created.Name)

		found, err := categoryRepository.FindOne(ctx, created.ID)
		assert.Nil(t, err)
		assert.Equal(t, created.Name, found.Name)
		assert.Equal(t, created.ID, found.ID)

		parent := found.Parent.Get()
		assert.Equal(t, deviceCategory.ID, parent.ID)
		assert.Equal(t, deviceCategory.Name, parent.Name)
		assert.Empty(t, parent.Parent.Get())
	})

	t.Run("update", func(t *testing.T) {
		ctx := context.Background()
		deviceCategory := domain.Category{
			Name: "device",
		}
		deviceCategory, _ = categoryRepository.Create(ctx, deviceCategory)

		appleCategory := domain.Category{
			Name: "apple",
		}
		appleCategory, _ = categoryRepository.Create(ctx, appleCategory)

		computerCategory := domain.Category{
			Name:   "computer",
			Parent: data.LazyLoadValue[domain.Category](deviceCategory),
		}
		computerCategory, _ = categoryRepository.Create(ctx, computerCategory)

		found, _ := categoryRepository.FindOne(ctx, computerCategory.ID)

		found.Name = "mac"
		found.Parent = data.LazyLoadValue[domain.Category](appleCategory)
		updated, err := categoryRepository.Update(ctx, found)
		assert.Nil(t, err)
		assert.Equal(t, found.Name, updated.Name)
		updatedParent := updated.Parent.Get()

		assert.Equal(t, appleCategory.ID, updatedParent.ID)
		assert.Equal(t, appleCategory.Name, updatedParent.Name)
	})

	t.Run("delete", func(t *testing.T) {
		ctx := context.Background()
		deviceCategory := domain.Category{
			Name: "device",
		}
		deviceCategory, _ = categoryRepository.Create(ctx, deviceCategory)

		computerCategory := domain.Category{
			Name:   "computer",
			Parent: data.LazyLoadValue[domain.Category](deviceCategory),
		}
		computerCategory, _ = categoryRepository.Create(ctx, computerCategory)

		err := categoryRepository.Delete(ctx, deviceCategory)
		assert.Nil(t, err)

		found, err := categoryRepository.FindOne(ctx, computerCategory.ID)
		assert.Nil(t, err)
		assert.Empty(t, found.Parent.Get().ID)

		err = categoryRepository.Delete(ctx, computerCategory)
		assert.Nil(t, err)
	})
}
