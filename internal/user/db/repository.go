package db

import (
	"errors"
)

var (
	ErrUserNotFound = errors.New("user not found")
	ErrBusinessProfileNotFound = errors.New("business profile not found")
)
