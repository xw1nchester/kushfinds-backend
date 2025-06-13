package db

import (
	"context"
	"time"
)

type Repository interface {
	CreateSession(ctx context.Context, token string, userAgent string, userID int, expiryDate time.Time) error
}
