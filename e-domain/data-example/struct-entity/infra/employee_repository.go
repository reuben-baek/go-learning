package infra

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
)

type EmployeeRepository struct {
	data.Repository[domain.Employee, uint]
	findByCompanyRepository    data.FindByRepository[domain.Employee, domain.Company]
	findByDepartmentRepository data.FindByRepository[domain.Employee, domain.Department]
}

func NewEmployeeRepository(
	repository data.Repository[domain.Employee, uint],
	findByCompanyRepository data.FindByRepository[domain.Employee, domain.Company],
	findByDepartmentRepository data.FindByRepository[domain.Employee, domain.Department],
) *EmployeeRepository {
	return &EmployeeRepository{Repository: repository, findByCompanyRepository: findByCompanyRepository, findByDepartmentRepository: findByDepartmentRepository}
}

func (e *EmployeeRepository) FindByCompany(ctx context.Context, company domain.Company) ([]domain.Employee, error) {
	return e.findByCompanyRepository.FindBy(ctx, "Company", company)
}

func (e *EmployeeRepository) FindByDepartment(ctx context.Context, department domain.Department) ([]domain.Employee, error) {
	return e.findByDepartmentRepository.FindBy(ctx, "Department", department)
}
