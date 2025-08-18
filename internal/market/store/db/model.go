package storedb

import "errors"

var (
	ErrStoreTypeNotFound = errors.New("store type not found")
	ErrStoreNotFound = errors.New("store not found")
)