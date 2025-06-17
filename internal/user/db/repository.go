package db

import (
	"context"
	"errors"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type Repository interface {
	GetByID(ctx context.Context, id int) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, email string) (int, error)
	Verify(ctx context.Context, id int) (*User, error)
	CheckUsernameIsAvailable(ctx context.Context, username string) (bool, error)
	SetProfileInfo(ctx context.Context, user User) (*User, error)
	SetPassword(ctx context.Context, id int, passwordHash []byte) error
}
