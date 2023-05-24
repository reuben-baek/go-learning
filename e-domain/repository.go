package e_domain

import "context"

type Repository[T any, I comparable] interface {
	FindOne(ctx context.Context, id I) (T, error)
	Create(ctx context.Context, o T) (T, error)
	Update(ctx context.Context, o T) (T, error)
	Delete(ctx context.Context, o T) error
}

type ServerRepository interface {
	Repository[Server, string]
}

type FlavorRepository interface {
	Repository[Flavor, string]
}
