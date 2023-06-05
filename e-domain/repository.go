package e_domain

import "context"

type Repository[T any, ID comparable] interface {
	FindOne(ctx context.Context, id ID) (T, error)
	FindBy(ctx context.Context, belongTo any) ([]T, error)
	Create(ctx context.Context, entity T) (T, error)
	Update(ctx context.Context, entity T) (T, error)
	Delete(ctx context.Context, entity T) error
}
