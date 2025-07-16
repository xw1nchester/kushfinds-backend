package handler

import "github.com/vetrovegor/kushfinds-backend/internal/location/country"

type ContriesResponse struct {
	Contries []country.Country `json:"countries"`
}
