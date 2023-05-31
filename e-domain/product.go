package e_domain

type Product struct {
	ID     uint
	Name   string
	Weight uint
}

type ProductRepository interface {
	Repository[Product, uint]
}
