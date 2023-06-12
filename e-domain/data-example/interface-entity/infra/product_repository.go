package infra

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/interface-entity/domain"
)

type ProductRepository struct {
	data.Repository[domain.Product, uint]
	data.BelongToRepository[domain.Product, domain.Company]
}

func NewProductRepository(repository data.Repository[domain.Product, uint], belongToRepository data.BelongToRepository[domain.Product, domain.Company]) *ProductRepository {
	return &ProductRepository{Repository: repository, BelongToRepository: belongToRepository}
}

func (p *ProductRepository) FindByCompany(ctx context.Context, company domain.Company) ([]domain.Product, error) {
	return p.FindBy(ctx, company)
}
