package f_repository_impl_test

import (
	"context"
	e_domain "github.com/reuben-baek/go-learning/e-domain"
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

type Product struct {
	ID     uint   `gorm:"primaryKey;column:id"`
	Name   string `gorm:"column:name"`
	Weight uint   `gorm:"column:weight"`
}

func (p Product) To() e_domain.Product {
	return e_domain.Product{
		ID:     p.ID,
		Name:   p.Name,
		Weight: p.Weight,
	}
}
func (p Product) From(m e_domain.Product) any {
	return Product{
		ID:     m.ID,
		Name:   m.Name,
		Weight: m.Weight,
	}
}

func NewProductRepository(dtoRepository e_domain.Repository[Product, uint]) e_domain.ProductRepository {
	return f_repository_impl.NewDtoWrapRepository[Product, e_domain.Product, uint](dtoRepository)
}

func NewGormProductRepository(db *gorm.DB) e_domain.ProductRepository {
	return f_repository_impl.NewGormDtoWrapRepository[Product, e_domain.Product, uint](db)
}

func NewInMemoryProductRepository() e_domain.ProductRepository {
	dtoRepository := f_repository_impl.NewInMemoryRepository[Product, uint]()
	return f_repository_impl.NewDtoWrapRepository[Product, e_domain.Product, uint](dtoRepository)
}

func TestProductRepository(t *testing.T) {
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

	db.AutoMigrate(&Product{})

	productRepository := NewGormProductRepository(db)

	ctx := context.Background()
	macM1 := e_domain.Product{
		Name:   "mac-m1",
		Weight: 1000,
	}
	created, err := productRepository.Create(ctx, macM1)
	assert.Nil(t, err)
	assert.NotEmpty(t, created.ID)

	found, err := productRepository.FindOne(ctx, created.ID)
	assert.Nil(t, err)
	assert.Equal(t, created.ID, found.ID)

	macM1Update := found
	macM1Update.Weight = 2000
	updated, err := productRepository.Update(ctx, macM1Update)
	assert.Nil(t, err)
	assert.Equal(t, macM1Update.ID, updated.ID)
	assert.Equal(t, macM1Update.Name, updated.Name)
	assert.Equal(t, macM1Update.Weight, updated.Weight)

	foundUpdated, err := productRepository.FindOne(ctx, macM1Update.ID)
	assert.Nil(t, err)
	assert.Equal(t, macM1Update.ID, foundUpdated.ID)
	assert.Equal(t, macM1Update.Name, foundUpdated.Name)
	assert.Equal(t, macM1Update.Weight, updated.Weight)

	err = productRepository.Delete(ctx, found)
	assert.Nil(t, err)

	_, err = productRepository.FindOne(ctx, found.ID)
	assert.ErrorIs(t, f_repository_impl.NotFoundError, err)

}
