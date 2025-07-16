package db

import "errors"

var (
	ErrStateNotFound = errors.New("state not found")
)

type State struct {
	ID   int
	Name string
}
