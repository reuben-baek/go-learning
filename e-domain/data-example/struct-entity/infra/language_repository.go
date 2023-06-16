package infra

import "github.com/reuben-baek/go-learning/e-domain/data"

type LanguageRepository struct {
	data.Repository[Language, string]
}

func NewLanguageRepository(repository data.Repository[Language, string]) *LanguageRepository {
	return &LanguageRepository{Repository: repository}
}
