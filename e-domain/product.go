package e_domain

import "context"

type Company struct {
	ID   uint
	Name string
}
type Product struct {
	ID      uint
	Name    string
	Weight  uint
	Company *LazyLoad[Company]
}

type ProductRepository interface {
	Repository[Product, uint]
	FindByCompany(ctx context.Context, company Company) ([]Product, error)
}
