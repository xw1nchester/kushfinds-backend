package handler

import (
	"github.com/xw1nchester/kushfinds-backend/internal/industry"
)

type IndustriesResponse struct {
	Industries []industry.Industry `json:"industries"`
}
