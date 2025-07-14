package db

import (
	"github.com/vetrovegor/kushfinds-backend/internal/location/country"
	"github.com/vetrovegor/kushfinds-backend/internal/location/region"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state"
)

type User struct {
	ID           int
	Email        string
	Username     *string
	FirstName    *string
	LastName     *string
	Avatar       *string
	PasswordHash *[]byte
	IsVerified   bool
	IsAdmin      bool
	Age          *int
	PhoneNumber  *string
	Country      *country.Country
	State        *state.State
	Region      *region.Region
}
