package brand

import (
	"time"

	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	marketsection "github.com/xw1nchester/kushfinds-backend/internal/market/section"
)

type Brand struct {
	ID                int                           `json:"id"`
	UserID            int                           `json:"-"`
	Country           country.Country               `json:"country"`
	MarketSection     marketsection.MarketSection   `json:"marketSection"`
	MarketSubSections []marketsection.MarketSection `json:"marketSections"`
	States            []state.State                 `json:"states"`
	Name              string                        `json:"name"`
	Email             string                        `json:"email"`
	PhoneNumber       string                        `json:"phoneNumber"`
	Logo              string                        `json:"logo"`
	Banner            string                        `json:"banner"`
	Documents         []string                      `json:"documents"`
	IsPublished       bool                          `json:"isPublished"`
	CreatedAt         time.Time                     `json:"createdAt"`
	UpdatedAt         time.Time                     `json:"updatedAt"`
}

type BrandSummary struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Logo string `json:"logo"`
}
