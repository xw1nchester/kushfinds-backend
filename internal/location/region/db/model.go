package db

import "errors"

var (
	ErrRegionNotFound   = errors.New("region not found")
	ErrLocationNotFound = errors.New("location not found")
)
