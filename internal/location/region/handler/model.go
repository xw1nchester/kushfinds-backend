package handler

import "github.com/vetrovegor/kushfinds-backend/internal/location/region"

type RegionsResponse struct {
	Regions []region.Region `json:"regions"`
}
