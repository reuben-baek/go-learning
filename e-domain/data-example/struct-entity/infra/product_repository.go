package infra

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
)

type ProductRepository struct {
	data.Repository[domain.Product, uint]
	companyBelongToRepository  data.FindByRepository[domain.Product, domain.Company]
	categoryBelongToRepository data.FindByRepository[domain.Product, domain.Category]
}

func NewProductRepository(
	repository data.Repository[domain.Product, uint],
	companyBelongToRepository data.FindByRepository[domain.Product, domain.Company],
	categoryBelongToRepository data.FindByRepository[domain.Product, domain.Category],
) *ProductRepository {
	return &ProductRepository{
		Repository:                 repository,
		companyBelongToRepository:  companyBelongToRepository,
		categoryBelongToRepository: categoryBelongToRepository,
	}
}

func (p *ProductRepository) FindByCompany(ctx context.Context, company domain.Company) ([]domain.Product, error) {
	return p.companyBelongToRepository.FindBy(ctx, "Company", company)
}

func (p *ProductRepository) FindByCategory(ctx context.Context, category domain.Category) ([]domain.Product, error) {
	return p.categoryBelongToRepository.FindBy(ctx, "Category", category)
}
