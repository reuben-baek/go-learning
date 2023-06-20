package infra

import (
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
)

type DepartmentRepository struct {
	data.Repository[domain.Department, uint]
}

func NewDepartmentRepository(repository data.Repository[domain.Department, uint]) *DepartmentRepository {
	return &DepartmentRepository{Repository: repository}
}
