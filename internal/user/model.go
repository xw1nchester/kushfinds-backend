package user

import (
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
)

type User struct {
	ID                 int              `json:"id"`
	Email              string           `json:"email"`
	Username           *string          `json:"username"`
	FirstName          *string          `json:"firstName"`
	LastName           *string          `json:"lastName"`
	Avatar             *string          `json:"avatar"`
	IsVerified         bool             `json:"isVerified"`
	PasswordHash       *[]byte          `json:"-"`
	IsPasswordSet      bool             `json:"isPasswordSet"`
	IsAdmin            bool             `json:"isAdmin"`
	Age                *int             `json:"age"`
	PhoneNumber        *string          `json:"phoneNumber"`
	Country            *country.Country `json:"country"`
	State              *state.State     `json:"state"`
	Region             *region.Region   `json:"region"`
	HasBusinessProfile bool             `json:"hasBusinessProfile"`
}

type UserResponse struct {
	User User `json:"user"`
}

type BusinessIndustry struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type BusinessProfile struct {
	UserID           int              `json:"-"`
	BusinessIndustry BusinessIndustry `json:"businessIndustry"`
	BusinessName     string           `json:"businessName"`
	Country          country.Country  `json:"country"`
	State            state.State      `json:"state"`
	Region           region.Region    `json:"region"`
	Email            string           `json:"email"`
	PhoneNumber      string           `json:"phoneNumber"`
	IsVerified       bool             `json:"isVerified"`
}

type BusinessProfileResponse struct {
	BusinessProfile *BusinessProfile `json:"businessProfile"`
}
