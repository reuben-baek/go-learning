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

func TestDepartmentRepository(t *testing.T) {
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

	transactionManager := data.NewGormTransactionManager(db)
	companyGormRepository := data.NewGormRepository[infra.Company, uint](transactionManager)
	departmentGormRepository := data.NewGormRepository[infra.Department, uint](transactionManager)

	companyRepository := data.NewDtoWrapRepository[infra.Company, domain.Company, uint](companyGormRepository)

	departmentRepository := infra.NewDepartmentRepository(
		data.NewDtoWrapRepository[infra.Department, domain.Department, uint](departmentGormRepository),
		data.NewDtoWrapFindByRepository[infra.Department, domain.Department, infra.Department, domain.Department](
			data.NewGormFindByRepository[infra.Department, infra.Department, uint](departmentGormRepository),
		),
	)

	ctx := context.Background()
	kakaoEnterprise := domain.Company{
		Name: "kakao enterprise",
	}
	kakaoEnterprise, _ = companyRepository.Create(ctx, kakaoEnterprise)
	kakaoCloud := domain.Company{
		Name: "kakao cloud",
	}
	kakaoCloud, _ = companyRepository.Create(ctx, kakaoCloud)

	cloudDevTeam := domain.Department{
		Name:    "cloud development team",
		Company: data.LazyLoadValue[domain.Company](kakaoEnterprise),
		Upper:   nil,
		Manager: nil,
	}
	cloudDevTeam, _ = departmentRepository.Create(ctx, cloudDevTeam)

	storageDevPart := domain.Department{
		Name:    "storage development part",
		Company: data.LazyLoadValue[domain.Company](kakaoEnterprise),
		Upper:   data.LazyLoadValue[domain.Department](cloudDevTeam),
		Manager: nil,
	}
	storageDevPart, _ = departmentRepository.Create(ctx, storageDevPart)

	t.Run("find-by-upper", func(t *testing.T) {
		departments, err := departmentRepository.FindByUpper(ctx, cloudDevTeam)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(departments))
		assert.Equal(t, storageDevPart.ID, departments[0].ID)
		assert.Equal(t, storageDevPart.Name, departments[0].Name)
	})
}
