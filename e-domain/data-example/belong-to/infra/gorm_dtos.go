package infra

import (
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/belong-to/domain"
)

type Company struct {
	ID   uint   `gorm:"primaryKey;column:id"`
	Name string `gorm:"column:name"`
}

func (c Company) To() domain.Company {
	return domain.Company{
		ID:   c.ID,
		Name: c.Name,
	}
}
func (c Company) From(m domain.Company) any {
	return Company{
		ID:   m.ID,
		Name: m.Name,
	}
}

type Product struct {
	data.LazyLoader `gorm:"-"`
	ID              uint   `gorm:"primaryKey;column:id"`
	Name            string `gorm:"column:name"`
	Weight          uint   `gorm:"column:weight"`
	CompanyID       uint
	Company         Company `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (p Product) To() domain.Product {
	return domain.Product{
		ID:     p.ID,
		Name:   p.Name,
		Weight: p.Weight,
		Company: data.LazyLoadFn[domain.Company](func() (any, error) {
			if company, err := data.LazyLoadNow[Company](&p); err != nil {
				return nil, err
			} else {
				return company.To(), nil
			}
		}),
	}
}
func (p Product) From(m domain.Product) any {
	return Product{
		ID:        m.ID,
		Name:      m.Name,
		Weight:    m.Weight,
		CompanyID: m.Company.Get().ID,
	}
}
