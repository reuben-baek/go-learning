package domain

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
)

type Company interface {
	ID() uint
	Name() string
}

type Product interface {
	ID() uint
	Name() string
	Weight() uint
	Company() *data.LazyLoad[Company]
}

type company struct {
	id   uint
	name string
}

func CompanyInstance(id uint, name string) Company {
	return &company{
		id:   id,
		name: name,
	}
}
func (c company) ID() uint {
	return c.id
}

func (c company) Name() string {
	return c.name
}

type product struct {
	id      uint
	name    string
	weight  uint
	company *data.LazyLoad[Company]
}

func ProductInstance(id uint, name string, weight uint, company *data.LazyLoad[Company]) Product {
	return &product{id: id, name: name, weight: weight, company: company}
}

func (p product) ID() uint {
	return p.id
}

func (p product) Name() string {
	return p.name
}

func (p product) Weight() uint {
	return p.weight
}

func (p product) Company() *data.LazyLoad[Company] {
	return p.company
}

// ProductRepository is an example of belong-to or many-to-one association and lazy loading.
// Refer its implementation and unit tests in infra/product_repository_test.go.
type ProductRepository interface {
	data.Repository[Product, uint]
	FindByCompany(ctx context.Context, company Company) ([]Product, error)
}
