package infra

import (
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
)

type CompanyRepository struct {
	data.Repository[domain.Company, uint]
}

func NewCompanyRepository(repository data.Repository[domain.Company, uint]) *CompanyRepository {
	return &CompanyRepository{Repository: repository}
}
