package domain

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
)

type Company struct {
	ID   uint
	Name string
}
type Product struct {
	ID      uint
	Name    string
	Weight  uint
	Company *data.LazyLoad[Company]
}

// ProductRepository is an example of belong-to or many-to-one association and lazy loading.
// Refer its implementation and unit tests in f-repository-impl/product_repository_test.go.
type ProductRepository interface {
	data.Repository[Product, uint]
	FindByCompany(ctx context.Context, company Company) ([]Product, error)
}
