package storehandler

import (
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	"github.com/xw1nchester/kushfinds-backend/internal/market/brand"
	"github.com/xw1nchester/kushfinds-backend/internal/market/store"
	"github.com/xw1nchester/kushfinds-backend/pkg/types"
)

type StoreTypesResponse struct {
	StoreTypes []store.StoreType `json:"storeTypes"`
}

type StoreRequest struct {
	BrandID           types.IntOrString `json:"brandId" validate:"required"`
	Name              string            `json:"name" validate:"required"`
	Banner            string            `json:"banner" validate:"required"`
	Description       string            `json:"description" validate:"required"`
	CountryID         types.IntOrString `json:"countryId" validate:"required"`
	StateID           types.IntOrString `json:"stateId" validate:"required"`
	RegionID          types.IntOrString `json:"regionId" validate:"required"`
	Street            string            `json:"street" validate:"required"`
	House             string            `json:"house" validate:"required"`
	PostCode          string            `json:"postCode" validate:"required"`
	Email             string            `json:"email" validate:"required,email"`
	PhoneNumber       string            `json:"phoneNumber" validate:"required"`
	StoreTypeID       types.IntOrString `json:"storeTypeId" validate:"required"`
	DeliveryPrice     types.IntOrString `json:"deliveryPrice" validate:"required"`
	MinimalOrderPrice types.IntOrString `json:"minimalOrderPrice" validate:"required"`
	DeliveryDistance  types.IntOrString `json:"deliveryDistance" validate:"required"`
	Pictures          []string          `json:"pictures" validate:"required"`
}

func (sr *StoreRequest) ToDomain(userID int) *store.Store {
	return &store.Store{
		UserID:            userID,
		Brand:             brand.BrandSummary{ID: int(sr.BrandID)},
		Name:              sr.Name,
		Banner:            sr.Banner,
		Description:       sr.Description,
		Country:           country.Country{ID: int(sr.CountryID)},
		State:             state.State{ID: int(sr.StateID)},
		Region:            region.Region{ID: int(sr.RegionID)},
		Street:            sr.Street,
		House:             sr.House,
		PostCode:          sr.PostCode,
		Email:             sr.Email,
		PhoneNumber:       sr.PhoneNumber,
		StoreType:         store.StoreType{ID: int(sr.StoreTypeID)},
		DeliveryPrice:     int(sr.DeliveryPrice),
		MinimalOrderPrice: int(sr.MinimalOrderPrice),
		DeliveryDistance:  int(sr.DeliveryDistance),
		Pictures:          sr.Pictures,
	}
}

type StoreResponse struct {
	Store store.Store `json:"store"`
}

func NewStoreResponse(s store.Store, staticURL string) StoreResponse {
	s.Banner = staticURL + "/" + s.Banner
	for i := range s.Pictures {
		s.Pictures[i] = staticURL + "/" + s.Pictures[i]
	}
	return StoreResponse{Store: s}
}

type StoresSummaryResponse struct {
	Stores []store.StoreSummary `json:"stores"`
}

func NewStoresSummaryResponse(elements []store.StoreSummary, staticURL string) StoresSummaryResponse {
	for i := range elements {
		elements[i].Banner = staticURL + "/" + elements[i].Banner
	}
	return StoresSummaryResponse{Stores: elements}
}