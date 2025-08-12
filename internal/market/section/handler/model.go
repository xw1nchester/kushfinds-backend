package handler

import (
	marketsection "github.com/xw1nchester/kushfinds-backend/internal/market/section"
)

type MarketSectionsResponse struct {
	MarketSections []marketsection.MarketSection `json:"marketSections"`
}
