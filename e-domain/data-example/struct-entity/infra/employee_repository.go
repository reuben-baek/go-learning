package infra

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
)

type EmployeeRepository struct {
	data.Repository[domain.Employee, uint]
	companyBelongToRepository data.BelongToRepository[domain.Employee, domain.Company]
}

func NewEmployeeRepository(repository data.Repository[domain.Employee, uint], companyBelongToRepository data.BelongToRepository[domain.Employee, domain.Company]) *EmployeeRepository {
	return &EmployeeRepository{Repository: repository, companyBelongToRepository: companyBelongToRepository}
}

func (e *EmployeeRepository) FindByCompany(ctx context.Context, company domain.Company) ([]domain.Employee, error) {
	return e.companyBelongToRepository.FindBy(ctx, company)
}
