package handler

import "github.com/vetrovegor/kushfinds-backend/pkg/types"

type ProfileRequest struct {
	FirstName   *string            `json:"firstName" validate:"omitempty,min=3,max=30"`
	LastName    *string            `json:"lastName" validate:"omitempty,min=3,max=30"`
	Age         *types.IntOrString `json:"age" validate:"omitempty,gt=0"`
	PhoneNumber *string            `json:"phoneNumber" validate:"omitempty"`
	CountryID   *types.IntOrString `json:"countryId" validate:"omitempty"`
	StateID     *types.IntOrString `json:"stateId" validate:"omitempty"`
	RegionID    *types.IntOrString `json:"regionId" validate:"omitempty"`
}
