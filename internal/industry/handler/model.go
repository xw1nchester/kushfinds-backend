package handler

import (
	"github.com/vetrovegor/kushfinds-backend/internal/industry"
)

type IndustriesResponse struct {
	Industries []industry.Industry `json:"industries"`
}
