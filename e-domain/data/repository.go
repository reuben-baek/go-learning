package data

import "context"

type Repository[T any, ID comparable] interface {
	FindOne(ctx context.Context, id ID) (T, error)
	Create(ctx context.Context, entity T) (T, error)
	Update(ctx context.Context, entity T) (T, error)
	Delete(ctx context.Context, entity T) error
}

type FindByRepository[T any, S any] interface {
	FindBy(ctx context.Context, name string, byEntity S) ([]T, error)
}
