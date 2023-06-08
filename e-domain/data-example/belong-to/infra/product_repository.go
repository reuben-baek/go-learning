package infra

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/belong-to/domain"
)

type ProductRepository struct {
	data.Repository[domain.Product, uint]
}

func NewProductRepository(repository data.Repository[domain.Product, uint]) *ProductRepository {
	return &ProductRepository{Repository: repository}
}

func (p *ProductRepository) FindByCompany(ctx context.Context, company domain.Company) ([]domain.Product, error) {
	return p.FindBy(ctx, company)
}
