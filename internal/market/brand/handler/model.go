package handler

import (
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	"github.com/xw1nchester/kushfinds-backend/internal/market/brand"
	marketsection "github.com/xw1nchester/kushfinds-backend/internal/market/section"
	"github.com/xw1nchester/kushfinds-backend/pkg/types"
	"github.com/xw1nchester/kushfinds-backend/pkg/utils"
)

type BrandRequest struct {
	CountryID           types.IntOrString   `json:"country" validate:"required"`
	MarketSection       types.IntOrString   `json:"marketSection" validate:"required"`
	MarketSubSectionIDs []types.IntOrString `json:"marketSubSectionIds" validate:"required,dive,gt=0"`
	StateIDs            []types.IntOrString `json:"stateIds" validate:"required,dive,gt=0"`
	Name                string              `json:"name" validate:"required"`
	Email               string              `json:"email" validate:"required,email"`
	PhoneNumber         string              `json:"phoneNumber" validate:"required"`
	Logo                string              `json:"logo" validate:"required"`
	Banner              string              `json:"banner" validate:"required"`
}

func (br *BrandRequest) ToDomain(userID int) *brand.Brand {
	var marketSubSections []marketsection.MarketSection
	for _, id := range utils.RemoveDuplicates(br.MarketSubSectionIDs) {
		marketSubSections = append(
			marketSubSections,
			marketsection.MarketSection{ID: int(id)},
		)
	}

	var states []state.State
	for _, id := range utils.RemoveDuplicates(br.StateIDs) {
		states = append(
			states,
			state.State{ID: int(id)},
		)
	}

	return &brand.Brand{
		UserID: userID,
		Country: country.Country{
			ID: int(br.CountryID),
		},
		MarketSection: marketsection.MarketSection{
			ID: int(br.MarketSection),
		},
		MarketSubSections: marketSubSections,
		States:            states,
		Name:              br.Name,
		Email:             br.Email,
		PhoneNumber:       br.PhoneNumber,
		Logo:              br.Logo,
		Banner:            br.Banner,
	}
}

type BrandResponse struct {
	Brand brand.Brand `json:"brand"`
}

func NewBrandResponse(b brand.Brand, staticURL string) BrandResponse {
	b.Logo = staticURL + "/" + b.Logo
	b.Banner = staticURL + "/" + b.Banner
	return BrandResponse{Brand: b}
}

type BrandsSummaryResponse struct {
	Brands []brand.BrandSummary `json:"brands"`
}

func NewBrandsSummaryResponse(bs []brand.BrandSummary, staticURL string) BrandsSummaryResponse {
	for i := range bs {
		bs[i].Logo = staticURL + "/" + bs[i].Logo
	}
	return BrandsSummaryResponse{Brands: bs}
}
