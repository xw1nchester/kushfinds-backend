package db

import (
	"context"
	"time"
)

//go:generate mockgen -source=repository.go -destination=mocks/mock.go -package=mockauthdb
type Repository interface {
	CreateSession(ctx context.Context, token string, userAgent string, userID int, expiryDate time.Time) error
	DeleteNotExpirySessionByToken(ctx context.Context, token string) (int, error)
}
