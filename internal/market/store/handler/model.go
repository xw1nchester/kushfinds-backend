package storehandler

import "github.com/xw1nchester/kushfinds-backend/internal/market/store"

type StoreTypesResponse struct {
	StoreTypes []store.StoreType `json:"storeTypes"`
}
