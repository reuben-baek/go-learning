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

type Company struct {
	ID   uint   `gorm:"primaryKey;column:id"`
	Name string `gorm:"column:name"`
}

func (c Company) To() e_domain.Company {
	return e_domain.Company{
		ID:   c.ID,
		Name: c.Name,
	}
}

type Product struct {
	f_repository_impl.LazyLoadableImpl `gorm:"-"`
	ID                                 uint   `gorm:"primaryKey;column:id"`
	Name                               string `gorm:"column:name"`
	Weight                             uint   `gorm:"column:weight"`
	CompanyID                          uint
	Company                            Company `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (p Product) To() e_domain.Product {
	return e_domain.Product{
		ID:     p.ID,
		Name:   p.Name,
		Weight: p.Weight,
		Company: e_domain.LazyLoadFn[e_domain.Company](func() (any, error) {
			if company, err := f_repository_impl.LazyLoadNow[Company](&p); err != nil {
				return nil, err
			} else {
				return company.To(), nil
			}
		}),
	}
}
func (p Product) From(m e_domain.Product) any {
	return Product{
		ID:        m.ID,
		Name:      m.Name,
		Weight:    m.Weight,
		CompanyID: m.Company.Get().ID,
	}
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
	db.AutoMigrate(&Company{})

	companyRepository := f_repository_impl.NewGormRepository[Company, int](db)

	ctx := context.Background()
	kakaoEnterprise := Company{
		Name: "kakao enterprise",
	}
	kakaoEnterpriseCreated, err := companyRepository.Create(ctx, kakaoEnterprise)
	assert.Nil(t, err)
	assert.NotEmpty(t, kakaoEnterpriseCreated.ID)

	productRepository := f_repository_impl.NewGormDtoWrapRepository[Product, e_domain.Product, uint](db)

	domainCompany := e_domain.Company{
		ID:   kakaoEnterpriseCreated.ID,
		Name: kakaoEnterpriseCreated.Name,
	}
	macM1 := e_domain.Product{
		Name:    "mac-m1",
		Weight:  1000,
		Company: e_domain.LazyLoadValue(domainCompany),
	}
	created, err := productRepository.Create(ctx, macM1)
	assert.Nil(t, err)
	assert.NotEmpty(t, created.ID)

	found, err := productRepository.FindOne(ctx, created.ID)
	assert.Nil(t, err)
	assert.Equal(t, created.ID, found.ID)
	company := found.Company.Get()
	assert.Equal(t, domainCompany, company)

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
