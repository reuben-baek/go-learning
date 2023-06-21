package infra

import (
	"context"
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
)

type DepartmentRepository struct {
	data.Repository[domain.Department, uint]
	findByUpperRepository data.FindByRepository[domain.Department, domain.Department]
}

func NewDepartmentRepository(repository data.Repository[domain.Department, uint], findByUpperRepository data.FindByRepository[domain.Department, domain.Department]) *DepartmentRepository {
	return &DepartmentRepository{Repository: repository, findByUpperRepository: findByUpperRepository}
}

func (e *DepartmentRepository) FindByUpper(ctx context.Context, upper domain.Department) ([]domain.Department, error) {
	return e.findByUpperRepository.FindBy(ctx, "Upper", upper)
}
