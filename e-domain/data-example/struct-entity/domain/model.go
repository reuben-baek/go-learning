package domain

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
)

type Company struct {
	ID   uint
	Name string
}

type Category struct {
	ID     uint
	Name   string
	Parent data.Lazy[Category] // belong-to self - self reference
}

type Product struct {
	ID       uint
	Name     string
	Category data.Lazy[Category] // belong-to
	Company  data.Lazy[Company]  // belong-to
}

type Employee struct {
	ID         uint
	Name       string
	Company    data.Lazy[Company]    // belong-to
	Manages    data.Lazy[[]Product]  // has-many
	CreditCard CreditCard            // has-one - eager
	Languages  data.Lazy[[]Language] // many-to-many
}

type CreditCard struct {
	ID     uint
	Number string
}

type Language struct {
	ID   string
	Name string
}

type CompanyRepository interface {
	data.Repository[Company, uint]
}
type CategoryRepository interface {
	data.Repository[Category, uint]
}

// ProductRepository is an example of belong-to or many-to-one association and lazy loading.
// Refer its implementation and unit tests in infra/product_repository_test.go.
type ProductRepository interface {
	data.Repository[Product, uint]
	FindByCompany(ctx context.Context, company Company) ([]Product, error)
	FindByCategory(ctx context.Context, category Category) ([]Product, error)
}

type EmployeeRepository interface {
	data.Repository[Employee, uint]
	FindByCompany(ctx context.Context, company Company) ([]Employee, error)
}

type LanguageRepository interface {
	data.Repository[Language, string]
}
