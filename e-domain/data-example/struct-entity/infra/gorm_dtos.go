package infra

import (
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
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
	c.ID = m.ID
	c.Name = m.Name
	return c
}

type Category struct {
	data.LazyLoader `gorm:"-"`
	ID              uint
	Name            string
	ParentID        *uint
	Parent          *Category `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
}

func (c Category) To() domain.Category {
	return domain.Category{
		ID:   c.ID,
		Name: c.Name,
		Parent: data.LazyLoadFn[domain.Category](func() (any, error) {
			if category, err := data.LazyLoadNow[*Category]("Parent", &c); err != nil {
				return nil, err
			} else {
				return category.To(), nil
			}
		}),
	}
}

func (c Category) From(m domain.Category) any {
	c.ID = m.ID
	c.Name = m.Name
	if m.Parent == nil {
		c.ParentID = nil
	} else {
		parentID := m.Parent.Get().ID
		c.ParentID = &parentID
	}
	return c
}

type Product struct {
	data.LazyLoader `gorm:"-"`
	ID              uint   `gorm:"primaryKey;column:id"`
	Name            string `gorm:"column:name"`
	CompanyID       uint
	Company         Company `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	CategoryID      uint
	Category        Category `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	EmployeeID      uint
}

func (p Product) To() domain.Product {
	return domain.Product{
		ID:   p.ID,
		Name: p.Name,
		Company: data.LazyLoadFn[domain.Company](func() (any, error) {
			if company, err := data.LazyLoadNow[Company]("Company", &p); err != nil {
				return nil, err
			} else {
				return company.To(), nil
			}
		}),
		Category: data.LazyLoadFn[domain.Category](func() (any, error) {
			if category, err := data.LazyLoadNow[Category]("Category", &p); err != nil {
				return nil, err
			} else {
				return category.To(), nil
			}
		}),
	}
}
func (p Product) From(m domain.Product) any {
	p.ID = m.ID
	p.Name = m.Name
	p.CompanyID = m.Company.Get().ID
	p.CategoryID = m.Category.Get().ID
	return p
}

type Employee struct {
	data.LazyLoader `gorm:"-"`
	ID              uint   `gorm:"primaryKey;column:id"`
	Name            string `gorm:"column:name"`
	CompanyID       uint
	Company         Company    `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	Manages         []Product  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	CreditCard      CreditCard `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;" fetch:"eager"`
	Languages       []Language `gorm:"many2many:employee_languages;" fetch:"lazy"`
}

func (e Employee) To() domain.Employee {
	return domain.Employee{
		ID:   e.ID,
		Name: e.Name,
		Company: data.LazyLoadFn[domain.Company](func() (any, error) {
			if company, err := data.LazyLoadNow[Company]("Company", &e); err != nil {
				return nil, err
			} else {
				return company.To(), nil
			}
		}),
		Manages: data.LazyLoadFn[[]domain.Product](func() (any, error) {
			if products, err := data.LazyLoadNow[[]Product]("Manages", &e); err != nil {
				return nil, err
			} else {
				ps := make([]domain.Product, 0, len(products))
				for _, p := range products {
					ps = append(ps, p.To())
				}
				return ps, nil
			}
		}),
		CreditCard: e.CreditCard.To(),
		Languages: data.LazyLoadFn[[]domain.Language](func() (any, error) {
			if languages, err := data.LazyLoadNow[[]Language]("Languages", &e); err != nil {
				return nil, err
			} else {
				ps := make([]domain.Language, 0, len(languages))
				for _, p := range languages {
					ps = append(ps, p.To())
				}
				return ps, nil
			}
		}),
	}
}

func (e Employee) From(m domain.Employee) any {
	var creditCard CreditCard
	manages := make([]Product, 0, len(m.Manages.Get()))
	languages := make([]Language, 0, len(m.Languages.Get()))

	for _, v := range m.Manages.Get() {
		var product Product
		product = product.From(v).(Product)
		manages = append(manages, product)
	}
	for _, v := range m.Languages.Get() {
		var language Language
		language = language.From(v).(Language)
		languages = append(languages, language)
	}

	creditCard.From(m.CreditCard)
	return Employee{
		ID:         m.ID,
		Name:       m.Name,
		CompanyID:  m.Company.Get().ID,
		Manages:    manages,
		CreditCard: creditCard,
		Languages:  languages,
	}
}

type CreditCard struct {
	ID         uint
	Number     string
	EmployeeID uint
}

func (c CreditCard) To() domain.CreditCard {
	return domain.CreditCard{
		ID:     c.ID,
		Number: c.Number,
	}
}

func (c CreditCard) From(m domain.CreditCard) any {
	c.ID = m.ID
	c.Number = m.Number
	return c
}

type Language struct {
	ID   string
	Name string
}

func (l Language) To() domain.Language {
	return domain.Language{
		ID:   l.ID,
		Name: l.Name,
	}
}

func (l Language) From(m domain.Language) any {
	l.ID = m.ID
	l.Name = m.Name
	return l
}
