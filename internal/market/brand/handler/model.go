package handler

import (
	"github.com/vetrovegor/kushfinds-backend/internal/location/country"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state"
	"github.com/vetrovegor/kushfinds-backend/internal/market/brand"
	marketsection "github.com/vetrovegor/kushfinds-backend/internal/market/section"
	"github.com/vetrovegor/kushfinds-backend/pkg/types"
)

type BrandRequest struct {
	CountryID     types.IntOrString `json:"country" validate:"required"`
	MarketSection types.IntOrString `json:"marketSection" validate:"required"`
	// возможно нужно использовать dive
	// TODO: подумать как оставлять только уникальные
	MarketSubSectionIDs []types.IntOrString `json:"marketSubSectionIds" validate:"required"`
	StateIDs            []types.IntOrString `json:"stateIds" validate:"required"`
	Name                string              `json:"name" validate:"required"`
	Email               string              `json:"email" validate:"required,email"`
	PhoneNumber         string              `json:"phoneNumber" validate:"required"`
	// TODO: validate url
	Logo   string `json:"logo" validate:"required"`
	Banner string `json:"banner" validate:"required"`
}

func (br *BrandRequest) ToDomain(userID int) *brand.Brand {
	var marketSubSections []marketsection.MarketSection
	for _, id := range br.MarketSubSectionIDs {
		marketSubSections = append(
			marketSubSections,
			marketsection.MarketSection{ID: int(id)},
		)
	}
	
	var states []state.State
	for _, id := range br.StateIDs {
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
