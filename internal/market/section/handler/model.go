package handler

import (
	marketsection "github.com/vetrovegor/kushfinds-backend/internal/market/section"
)

type MarketSectionsResponse struct {
	MarketSections []marketsection.MarketSection `json:"marketSections"`
}
