package db

import "errors"

var (
	ErrCountryNotFound = errors.New("country not found")
)

type Country struct {
	ID   int
	Name string
}
