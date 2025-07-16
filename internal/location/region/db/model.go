package db

import "errors"

var (
	ErrRegionNotFound = errors.New("region not found")
)

type Region struct {
	ID   int
	Name string
}
