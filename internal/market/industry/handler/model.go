package handler

import (
	"github.com/xw1nchester/kushfinds-backend/internal/market/industry"
)

type IndustriesResponse struct {
	Industries []industry.Industry `json:"industries"`
}
