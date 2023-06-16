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

func TestEmployeeRepository(t *testing.T) {
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

	db.AutoMigrate(&infra.Language{})
	db.AutoMigrate(&infra.CreditCard{})
	db.AutoMigrate(&infra.Category{})
	db.AutoMigrate(&infra.Product{})
	db.AutoMigrate(&infra.Company{})
	db.AutoMigrate(&infra.Employee{})

	companyGormRepository := data.NewGormRepository[infra.Company, uint](db)
	productGormRepository := data.NewGormRepository[infra.Product, uint](db)
	categoryGormRepository := data.NewGormRepository[infra.Category, uint](db)
	languageGormRepository := data.NewGormRepository[infra.Language, string](db)

	companyRepository := data.NewDtoWrapRepository[infra.Company, domain.Company, uint](companyGormRepository)
	categoryRepository := data.NewDtoWrapRepository[infra.Category, domain.Category, uint](categoryGormRepository)
	langusageRepository := data.NewDtoWrapRepository[infra.Language, domain.Language, string](languageGormRepository)

	productRepository := infra.NewProductRepository(
		data.NewDtoWrapRepository[infra.Product, domain.Product, uint](productGormRepository),
		data.NewDtoWrapBelongToRepository[infra.Product, domain.Product, infra.Company, domain.Company](
			data.NewGormBelongToRepository[infra.Product, infra.Company, uint](productGormRepository),
		),
		data.NewDtoWrapBelongToRepository[infra.Product, domain.Product, infra.Category, domain.Category](
			data.NewGormBelongToRepository[infra.Product, infra.Category, uint](productGormRepository),
		),
	)

	employeeGormRepository := data.NewGormRepository[infra.Employee, uint](db)
	employeeRepository := infra.NewEmployeeRepository(
		data.NewDtoWrapRepository[infra.Employee, domain.Employee, uint](employeeGormRepository),
		data.NewDtoWrapBelongToRepository[infra.Employee, domain.Employee, infra.Company, domain.Company](
			data.NewGormBelongToRepository[infra.Employee, infra.Company, uint](employeeGormRepository),
		),
	)

	ctx := context.Background()
	kakaoEnterprise := domain.Company{
		Name: "kakao enterprise",
	}
	kakaoEnterprise, err = companyRepository.Create(ctx, kakaoEnterprise)
	kakaoCloud := domain.Company{
		Name: "kakao cloud",
	}
	kakaoCloud, err = companyRepository.Create(ctx, kakaoCloud)

	cloud := domain.Category{
		Name: "cloud",
	}
	cloud, err = categoryRepository.Create(ctx, cloud)

	objectstorage := domain.Product{
		Name:     "object-storage",
		Category: data.LazyLoadValue[domain.Category](cloud),
		Company:  data.LazyLoadValue[domain.Company](kakaoEnterprise),
	}
	objectstorage, _ = productRepository.Create(ctx, objectstorage)

	korean := domain.Language{
		ID:   "kr",
		Name: "korean",
	}
	korean, _ = langusageRepository.Create(ctx, korean)

	t.Run("create & find", func(t *testing.T) {
		reuben := domain.Employee{
			Name:    "reuben.b",
			Company: data.LazyLoadValue[domain.Company](kakaoEnterprise),
			Manages: data.LazyLoadValue[[]domain.Product]([]domain.Product{objectstorage}),
			CreditCard: domain.CreditCard{
				Number: "111111111111",
			},
			Languages: data.LazyLoadValue[[]domain.Language]([]domain.Language{korean}),
		}
		reuben, err := employeeRepository.Create(ctx, reuben)
		assert.Nil(t, err)
		assert.NotEmpty(t, reuben.ID)

		t.Run("find-one", func(t *testing.T) {
			found, err := employeeRepository.FindOne(ctx, reuben.ID)
			assert.Nil(t, err)
			assert.Equal(t, reuben.ID, found.ID)
			assert.Equal(t, reuben.CreditCard, found.CreditCard)

			company := found.Company.Get()
			assert.Equal(t, kakaoEnterprise, company)
			manages := found.Manages.Get()
			assert.Equal(t, 1, len(manages))
			for _, v := range manages {
				category := v.Category.Get()
				assert.NotEmpty(t, category)
				company := v.Company.Get()
				assert.NotEmpty(t, company)
			}
			languages := found.Languages.Get()
			assert.Equal(t, 1, len(languages))
		})
		t.Run("find-by-company", func(t *testing.T) {
			employees, err := employeeRepository.FindByCompany(ctx, kakaoEnterprise)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(employees))
			assert.Equal(t, reuben.ID, employees[0].ID)
			assert.Equal(t, kakaoEnterprise, employees[0].Company.Get())
		})
	})

	t.Run("update", func(t *testing.T) {
	})

	t.Run("delete product", func(t *testing.T) {
	})

}
