package store

import (
	"time"

	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	"github.com/xw1nchester/kushfinds-backend/internal/market/brand"
	"github.com/xw1nchester/kushfinds-backend/internal/market/social"
)

type StoreType struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type Store struct {
	ID                int                   `json:"id"`
	UserID            int                   `json:"-"`
	Brand             brand.BrandSummary    `json:"brand"`
	Name              string                `json:"name"`
	Banner            string                `json:"banner"`
	Description       string                `json:"description"`
	Country           country.Country       `json:"country"`
	State             state.State           `json:"state"`
	Region            region.Region         `json:"region"`
	Street            string                `json:"street"`
	House             string                `json:"house"`
	PostCode          string                `json:"postCode"`
	Email             string                `json:"email"`
	PhoneNumber       string                `json:"phoneNumber"`
	StoreType         StoreType             `json:"storeType"`
	DeliveryPrice     int                   `json:"deliveryPrice"`
	MinimalOrderPrice int                   `json:"minimalOrderPrice"`
	DeliveryDistance  int                   `json:"deliveryDistance"`
	Pictures          []string              `json:"pictures"`
	Socials           []social.EntitySocial `json:"socials"`
	IsPublished       bool                  `json:"isPublished"`
	CreatedAt         time.Time             `json:"createdAt"`
	UpdatedAt         time.Time             `json:"updatedAt"`
}

type StoreSummary struct {
	ID     int                `json:"id"`
	Name   string             `json:"name"`
	Banner string             `json:"banner"`
	Brand  brand.BrandSummary `json:"brand"`
}
