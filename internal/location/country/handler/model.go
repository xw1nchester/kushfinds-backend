package handler

import "github.com/xw1nchester/kushfinds-backend/internal/location/country"

type ContriesResponse struct {
	Contries []country.Country `json:"countries"`
}
