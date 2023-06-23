package data

import "context"

type TransactionManager interface {
	Do(ctx context.Context, f func(ctx context.Context) error) error
	Get(ctx context.Context) any
}
