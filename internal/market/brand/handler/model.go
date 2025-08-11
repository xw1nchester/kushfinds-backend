package handler

import (
	"github.com/vetrovegor/kushfinds-backend/internal/location/country"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state"
	"github.com/vetrovegor/kushfinds-backend/internal/market/brand"
	marketsection "github.com/vetrovegor/kushfinds-backend/internal/market/section"
	"github.com/vetrovegor/kushfinds-backend/pkg/types"
	"github.com/vetrovegor/kushfinds-backend/pkg/utils"
)

type BrandRequest struct {
	CountryID     types.IntOrString `json:"country" validate:"required"`
	MarketSection types.IntOrString `json:"marketSection" validate:"required"`
	MarketSubSectionIDs []types.IntOrString `json:"marketSubSectionIds" validate:"required,dive,gt=0"`
	StateIDs            []types.IntOrString `json:"stateIds" validate:"required,dive,gt=0"`
	Name                string              `json:"name" validate:"required"`
	Email               string              `json:"email" validate:"required,email"`
	PhoneNumber         string              `json:"phoneNumber" validate:"required"`
	Logo   string `json:"logo" validate:"required,url"`
	Banner string `json:"banner" validate:"required,url"`
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
		States: states,
		Name: br.Name,
		Email: br.Email,
		PhoneNumber: br.PhoneNumber,
		Logo: br.Logo,
		Banner: br.Banner,
	}
}

type BrandResponse struct {
	Brand brand.Brand `json:"brand"`
}

type BrandsSummaryResponse struct {
	Brands []brand.BrandSummary `json:"brands"`
}
