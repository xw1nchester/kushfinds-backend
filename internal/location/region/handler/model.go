package handler

import "github.com/xw1nchester/kushfinds-backend/internal/location/region"

type RegionsResponse struct {
	Regions []region.Region `json:"regions"`
}
