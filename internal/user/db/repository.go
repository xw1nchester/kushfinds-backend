package db

import (
	"context"
	"errors"

	"github.com/vetrovegor/kushfinds-backend/internal/user"
)

var (
	ErrUserNotFound = errors.New("user not found")
)

type Repository interface {
	GetByID(ctx context.Context, id int) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	Create(ctx context.Context, email string) (int, error)
	Verify(ctx context.Context, id int) (*user.User, error)
	CheckUsernameIsAvailable(ctx context.Context, username string) (bool, error)
	SetProfileInfo(ctx context.Context, data user.User) (*user.User, error)
	SetPassword(ctx context.Context, id int, passwordHash []byte) error
}
