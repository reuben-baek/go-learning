package infra_test

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/belong-to/domain"
	"github.com/reuben-baek/go-learning/e-domain/data-example/belong-to/infra"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"os"
	"testing"
	"time"
)

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

	db.AutoMigrate(&infra.Product{})
	db.AutoMigrate(&infra.Company{})

	companyRepository := data.NewGormDtoWrapRepository[infra.Company, domain.Company, uint](db)
	productRepository := infra.NewProductRepository(data.NewGormDtoWrapRepository[infra.Product, domain.Product, uint](db))

	ctx := context.Background()
	kakaoEnterprise := domain.Company{
		Name: "kakao enterprise",
	}
	kakaoEnterprise, err = companyRepository.Create(ctx, kakaoEnterprise)
	kakaoCloud := domain.Company{
		Name: "kakao cloud",
	}
	kakaoCloud, err = companyRepository.Create(ctx, kakaoCloud)

	t.Run("create & find", func(t *testing.T) {
		macM1 := domain.Product{
			Name:    "macM1",
			Weight:  1000,
			Company: data.LazyLoadValue(kakaoEnterprise),
		}
		macM1, err = productRepository.Create(ctx, macM1)
		assert.Nil(t, err)
		assert.NotEmpty(t, macM1.ID)

		t.Run("find-one", func(t *testing.T) {
			found, err := productRepository.FindOne(ctx, macM1.ID)
			assert.Nil(t, err)
			assert.Equal(t, macM1.ID, found.ID)

			company := found.Company.Get()
			assert.Equal(t, kakaoEnterprise, company)
		})
		t.Run("find-by-company", func(t *testing.T) {
			products, err := productRepository.FindByCompany(ctx, kakaoEnterprise)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(products))
			assert.Equal(t, macM1.ID, products[0].ID)
			assert.Equal(t, kakaoEnterprise, products[0].Company.Get())
		})
	})

	t.Run("update product name", func(t *testing.T) {
		bareMetal := domain.Product{
			Name:    "bare metal",
			Weight:  1000,
			Company: data.LazyLoadValue(kakaoEnterprise),
		}
		bareMetal, err = productRepository.Create(ctx, bareMetal)
		assert.Nil(t, err)
		assert.NotEmpty(t, bareMetal.ID)

		found, _ := productRepository.FindOne(ctx, bareMetal.ID)

		found.Name = "base metal 2023"
		updated, err := productRepository.Update(ctx, found)
		assert.Nil(t, err)
		assert.Equal(t, bareMetal.ID, updated.ID)
		company := found.Company.Get()
		assert.Equal(t, kakaoEnterprise, company)

		foundAfterUpdate, err := productRepository.FindOne(ctx, bareMetal.ID)
		assert.Nil(t, err)
		assert.Equal(t, bareMetal.ID, foundAfterUpdate.ID)
		company = foundAfterUpdate.Company.Get()
		assert.Equal(t, kakaoEnterprise, company)
	})

	t.Run("update product company", func(t *testing.T) {
		bareMetal := domain.Product{
			Name:    "bare metal",
			Weight:  1000,
			Company: data.LazyLoadValue(kakaoEnterprise),
		}
		bareMetal, err = productRepository.Create(ctx, bareMetal)
		assert.Nil(t, err)
		assert.NotEmpty(t, bareMetal.ID)

		found, _ := productRepository.FindOne(ctx, bareMetal.ID)

		found.Company = data.LazyLoadValue[domain.Company](kakaoCloud)
		updated, err := productRepository.Update(ctx, found)
		assert.Nil(t, err)
		assert.Equal(t, bareMetal.ID, updated.ID)
		company := found.Company.Get()
		assert.Equal(t, kakaoCloud, company)

		foundAfterUpdate, err := productRepository.FindOne(ctx, bareMetal.ID)
		assert.Nil(t, err)
		assert.Equal(t, bareMetal.ID, foundAfterUpdate.ID)
		company = foundAfterUpdate.Company.Get()
		assert.Equal(t, kakaoCloud, company)
	})

	t.Run("delete product", func(t *testing.T) {
		bareMetal := domain.Product{
			Name:    "bare metal",
			Weight:  1000,
			Company: data.LazyLoadValue(kakaoEnterprise),
		}
		bareMetal, err = productRepository.Create(ctx, bareMetal)
		assert.Nil(t, err)
		assert.NotEmpty(t, bareMetal.ID)

		found, _ := productRepository.FindOne(ctx, bareMetal.ID)

		err := productRepository.Delete(ctx, found)
		assert.Nil(t, err)

		_, err = productRepository.FindOne(ctx, bareMetal.ID)
		assert.ErrorIs(t, data.NotFoundError, err)
	})
}
