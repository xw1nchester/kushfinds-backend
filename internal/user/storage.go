package user

import (
	"context"
)

type Repository interface {
	GetByID(ctx context.Context, id int) (User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, user User) (*User, error)
}