package user

import (
	"github.com/vetrovegor/kushfinds-backend/internal/location/country"
	"github.com/vetrovegor/kushfinds-backend/internal/location/region"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state"
)

type User struct {
	ID            int              `json:"id"`
	Email         string           `json:"email"`
	Username      *string          `json:"username"`
	FirstName     *string          `json:"firstName"`
	LastName      *string          `json:"lastName"`
	Avatar        *string          `json:"avatar"`
	IsVerified    bool             `json:"isVerified"`
	PasswordHash  *[]byte          `json:"-"`
	IsPasswordSet bool             `json:"isPasswordSet"`
	IsAdmin       bool             `json:"isAdmin"`
	Age           *int             `json:"age"`
	PhoneNumber   *string          `json:"phoneNumber"`
	Country       *country.Country `json:"country"`
	State         *state.State     `json:"state"`
	Region        *region.Region   `json:"region"`
}

type UserResponse struct {
	User User `json:"user"`
}
