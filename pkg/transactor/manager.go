package transactor

import "context"

type Manager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
