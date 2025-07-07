package db

import (
	"errors"
)

var (
	ErrCodeAlreadySent = errors.New("code has already been sent")
	ErrCodeNotFound = errors.New("code not found")
)
