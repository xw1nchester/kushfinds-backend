package handler

import "github.com/xw1nchester/kushfinds-backend/pkg/types"

type ProfileRequest struct {
	FirstName   *string            `json:"firstName" validate:"omitempty,min=3,max=30"`
	LastName    *string            `json:"lastName" validate:"omitempty,min=3,max=30"`
	Age         *types.IntOrString `json:"age" validate:"omitempty,gt=0"`
	PhoneNumber *string            `json:"phoneNumber" validate:"omitempty"`
	CountryID   *types.IntOrString `json:"countryId" validate:"omitempty"`
	StateID     *types.IntOrString `json:"stateId" validate:"omitempty"`
	RegionID    *types.IntOrString `json:"regionId" validate:"omitempty"`
}

type BusinessProfileRequest struct {
	BusinessIndustryID types.IntOrString `json:"businessIndustryId" validate:"required"`
	BusinessName       string            `json:"businessName" validate:"required,min=3,max=30"`
	CountryID          types.IntOrString `json:"countryId" validate:"required"`
	StateID            types.IntOrString `json:"stateId" validate:"required"`
	RegionID           types.IntOrString `json:"regionId" validate:"required"`
	Email              string            `json:"email" validate:"required,email"`
	PhoneNumber        string            `json:"phoneNumber" validate:"required"`
}

type AdminBusinessProfileRequest struct {
	BusinessProfileRequest
	// TODO: если установить validate:"required" и отправить false, то не пройдет валидацию
	IsVerified bool `json:"isVerified"`
}
