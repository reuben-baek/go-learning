package e_domain

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
}
