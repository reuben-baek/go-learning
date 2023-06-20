package infra

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
)

type EmployeeRepository struct {
	data.Repository[domain.Employee, uint]
	companyBelongToRepository    data.BelongToRepository[domain.Employee, domain.Company]
	departmentBelongToRepository data.BelongToRepository[domain.Employee, domain.Department]
}

func NewEmployeeRepository(
	repository data.Repository[domain.Employee, uint],
	companyBelongToRepository data.BelongToRepository[domain.Employee, domain.Company],
	departmentBelongToRepository data.BelongToRepository[domain.Employee, domain.Department],
) *EmployeeRepository {
	return &EmployeeRepository{Repository: repository, companyBelongToRepository: companyBelongToRepository, departmentBelongToRepository: departmentBelongToRepository}
}

func (e *EmployeeRepository) FindByCompany(ctx context.Context, company domain.Company) ([]domain.Employee, error) {
	return e.companyBelongToRepository.FindBy(ctx, company)
}

func (e *EmployeeRepository) FindByDepartment(ctx context.Context, department domain.Department) ([]domain.Employee, error) {
	// TODO
	panic("implement me")
}
