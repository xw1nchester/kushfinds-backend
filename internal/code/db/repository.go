package db

import (
	"context"
	"errors"
	"time"
)

var (
	ErrCodeAlreadySent = errors.New("code has already been sent")
)

type Repository interface {
	CheckNotExpiryCodeExists(ctx context.Context, codeType string, userID int) error
	Create(ctx context.Context, code string, codeType string, userID int, retryDate time.Time, expiryDate time.Time) error
	GetNotExpiryCode(ctx context.Context, code string, codeType string, userID int) error
}
