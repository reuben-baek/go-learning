package infra_test

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/infra"
	"github.com/sirupsen/logrus"
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
	departmentGormRepository := data.NewGormRepository[infra.Department, uint](db)

	companyRepository := data.NewDtoWrapRepository[infra.Company, domain.Company, uint](companyGormRepository)
	categoryRepository := data.NewDtoWrapRepository[infra.Category, domain.Category, uint](categoryGormRepository)
	languageRepository := data.NewDtoWrapRepository[infra.Language, domain.Language, string](languageGormRepository)

	productRepository := infra.NewProductRepository(
		data.NewDtoWrapRepository[infra.Product, domain.Product, uint](productGormRepository),
		data.NewDtoWrapFindByRepository[infra.Product, domain.Product, infra.Company, domain.Company](
			data.NewGormFindByRepository[infra.Product, infra.Company, uint](productGormRepository),
		),
		data.NewDtoWrapFindByRepository[infra.Product, domain.Product, infra.Category, domain.Category](
			data.NewGormFindByRepository[infra.Product, infra.Category, uint](productGormRepository),
		),
	)

	departmentRepository := infra.NewDepartmentRepository(
		data.NewDtoWrapRepository[infra.Department, domain.Department, uint](departmentGormRepository),
		data.NewDtoWrapFindByRepository[infra.Department, domain.Department, infra.Department, domain.Department](
			data.NewGormFindByRepository[infra.Department, infra.Department, uint](departmentGormRepository),
		),
	)
	employeeGormRepository := data.NewGormRepository[infra.Employee, uint](db)
	employeeRepository := infra.NewEmployeeRepository(
		data.NewDtoWrapRepository[infra.Employee, domain.Employee, uint](employeeGormRepository),
		data.NewDtoWrapFindByRepository[infra.Employee, domain.Employee, infra.Company, domain.Company](
			data.NewGormFindByRepository[infra.Employee, infra.Company, uint](employeeGormRepository),
		),
		data.NewDtoWrapFindByRepository[infra.Employee, domain.Employee, infra.Department, domain.Department](
			data.NewGormFindByRepository[infra.Employee, infra.Department, uint](employeeGormRepository),
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

	korean := domain.Language{
		ID:   "kr",
		Name: "korean",
	}
	korean, _ = languageRepository.Create(ctx, korean)

	t.Run("create & find", func(t *testing.T) {
		reuben := domain.Employee{
			Name:    "reuben.b",
			Company: data.LazyLoadValue[domain.Company](kakaoEnterprise),
			Manages: data.LazyLoadValue[[]domain.Product]([]domain.Product{objectstorage}),
			CreditCard: domain.CreditCard{
				Number: "111111111111",
			},
			Departments: data.LazyLoadValue[[]domain.Department]([]domain.Department{storageDevPart}),
			Languages:   data.LazyLoadValue[[]domain.Language]([]domain.Language{korean}),
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
			departments := found.Departments.Get()
			assert.Equal(t, 1, len(departments))
			assert.Equal(t, kakaoEnterprise, departments[0].Company.Get())
			upperDepartment := departments[0].Upper.Get()
			assert.Equal(t, cloudDevTeam.ID, upperDepartment.ID)
			assert.Equal(t, cloudDevTeam.Name, upperDepartment.Name)

			languages := found.Languages.Get()
			assert.Equal(t, 1, len(languages))
		})
		t.Run("find-by-company", func(t *testing.T) {
			employees, err := employeeRepository.FindByCompany(ctx, kakaoEnterprise)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(employees))
			assert.Equal(t, reuben.ID, employees[0].ID)

			employee := employees[0]
			company := employee.Company.Get()
			assert.Equal(t, kakaoEnterprise, company)
			manages := employee.Manages.Get()
			assert.Equal(t, 1, len(manages))
			for _, v := range manages {
				category := v.Category.Get()
				assert.NotEmpty(t, category)
				company := v.Company.Get()
				assert.NotEmpty(t, company)
			}

			departments := employee.Departments.Get()
			assert.Equal(t, 1, len(departments))
			assert.Equal(t, kakaoEnterprise, departments[0].Company.Get())
			upperDepartment := departments[0].Upper.Get()
			assert.Equal(t, cloudDevTeam.ID, upperDepartment.ID)
			assert.Equal(t, cloudDevTeam.Name, upperDepartment.Name)

			languages := employees[0].Languages.Get()
			assert.Equal(t, 1, len(languages))
		})
		t.Run("find-by-department", func(t *testing.T) {
			employees, err := employeeRepository.FindByDepartment(ctx, storageDevPart)
			assert.Nil(t, err)
			assert.Equal(t, 1, len(employees))
			assert.Equal(t, reuben.ID, employees[0].ID)

			employee := employees[0]
			company := employee.Company.Get()
			assert.Equal(t, kakaoEnterprise, company)
			manages := employee.Manages.Get()
			assert.Equal(t, 1, len(manages))
			for _, v := range manages {
				category := v.Category.Get()
				assert.NotEmpty(t, category)
				company := v.Company.Get()
				assert.NotEmpty(t, company)
			}

			departments := employee.Departments.Get()
			assert.Equal(t, 1, len(departments))
			assert.Equal(t, kakaoEnterprise, departments[0].Company.Get())
			upperDepartment := departments[0].Upper.Get()
			assert.Equal(t, cloudDevTeam.ID, upperDepartment.ID)
			assert.Equal(t, cloudDevTeam.Name, upperDepartment.Name)

			languages := employees[0].Languages.Get()
			assert.Equal(t, 1, len(languages))
		})
	})

	t.Run("update", func(t *testing.T) {
		reuben := domain.Employee{
			Name:    "reuben.b",
			Company: data.LazyLoadValue[domain.Company](kakaoEnterprise),
			Manages: data.LazyLoadValue[[]domain.Product]([]domain.Product{objectstorage}),
			CreditCard: domain.CreditCard{
				Number: "111111111111",
			},
			Departments: data.LazyLoadValue[[]domain.Department]([]domain.Department{storageDevPart}),
			Languages:   data.LazyLoadValue[[]domain.Language]([]domain.Language{korean}),
		}
		reuben, err := employeeRepository.Create(ctx, reuben)
		assert.Nil(t, err)
		assert.NotEmpty(t, reuben.ID)
		t.Run("name", func(t *testing.T) {
			found, _ := employeeRepository.FindOne(ctx, reuben.ID)

			found.Name = "reuben.baek"
			updated, err := employeeRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, updated.ID, found.ID)
			assert.Equal(t, updated.Name, found.Name)
			assert.Equal(t, kakaoEnterprise, updated.Company.Get())
			assert.Equal(t, 1, len(updated.Manages.Get()))
			assert.NotEmpty(t, updated.CreditCard)
			assert.Equal(t, 1, len(updated.Departments.Get()))
			assert.Equal(t, 1, len(updated.Languages.Get()))
		})
		t.Run("company", func(t *testing.T) {
			found, _ := employeeRepository.FindOne(ctx, reuben.ID)
			found.Company = data.LazyLoadValue[domain.Company](kakaoCloud)
			updated, err := employeeRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, kakaoCloud, updated.Company.Get())
		})
		t.Run("departments", func(t *testing.T) {
			found, _ := employeeRepository.FindOne(ctx, reuben.ID)
			found.Departments = data.LazyLoadValue[[]domain.Department]([]domain.Department{})
			updated, err := employeeRepository.Update(ctx, found)
			assert.Nil(t, err)
			assert.Equal(t, 0, len(updated.Departments.Get()))
		})
	})

	t.Run("delete product", func(t *testing.T) {
		reuben := domain.Employee{
			Name:    "reuben.b",
			Company: data.LazyLoadValue[domain.Company](kakaoEnterprise),
			Manages: data.LazyLoadValue[[]domain.Product]([]domain.Product{objectstorage}),
			CreditCard: domain.CreditCard{
				Number: "111111111111",
			},
			Departments: data.LazyLoadValue[[]domain.Department]([]domain.Department{storageDevPart}),
			Languages:   data.LazyLoadValue[[]domain.Language]([]domain.Language{korean}),
		}
		reuben, err := employeeRepository.Create(ctx, reuben)
		assert.Nil(t, err)
		assert.NotEmpty(t, reuben.ID)

		err = employeeRepository.Delete(ctx, reuben)
		assert.Nil(t, err)
		_, err = employeeRepository.FindOne(ctx, reuben.ID)
		assert.ErrorIs(t, data.NotFoundError, err)
	})

}

func init() {
	logrus.SetLevel(logrus.DebugLevel)
}
