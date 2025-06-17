package db

import (
	"context"
	"errors"
	"time"
)

var (
	ErrCodeAlreadySent = errors.New("code has already been sent")
	ErrCodeNotFound = errors.New("code not found")
)

type Repository interface {
	Create(ctx context.Context, code string, codeType string, userID int, retryDate time.Time, expiryDate time.Time) error
	CheckRecentlyCodeExists(ctx context.Context, codeType string, userID int) (bool, error)
	CheckNotExpiryCodeExists(ctx context.Context, code string, codeType string, userID int) (bool, error)
}
