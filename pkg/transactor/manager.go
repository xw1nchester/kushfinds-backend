package transactor

import "context"

//go:generate mockgen -source=manager.go -destination=mocks/mock.go -package=mocktransactor
type Manager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
