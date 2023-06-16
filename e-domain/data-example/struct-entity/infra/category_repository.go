package infra

import (
	"github.com/reuben-baek/go-learning/e-domain/data"
	"github.com/reuben-baek/go-learning/e-domain/data-example/struct-entity/domain"
)

type CategoryRepository struct {
	data.Repository[domain.Category, uint]
}

func NewCategoryRepository(repository data.Repository[domain.Category, uint]) *CategoryRepository {
	return &CategoryRepository{Repository: repository}
}
