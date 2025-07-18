package db

import (
	"github.com/vetrovegor/kushfinds-backend/internal/location/country"
	"github.com/vetrovegor/kushfinds-backend/internal/location/region"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
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
	Region       *region.Region
}

type BusinessIndustry struct {
	ID   int
	Name string
}

type BusinessProfile struct {
	UserID           int
	BusinessIndustry BusinessIndustry
	BusinessName     string
	Country          country.Country
	State            state.State
	Region           region.Region
	Email            string
	PhoneNumber      string
}

func (bp *BusinessProfile) ToDomain() *user.BusinessProfile {
	if bp == nil {
		return nil
	}
	
	return &user.BusinessProfile{
		BusinessIndustry: user.BusinessIndustry{
			ID:   bp.BusinessIndustry.ID,
			Name: bp.BusinessIndustry.Name,
		},
		BusinessName: bp.BusinessName,
		Country:      bp.Country,
		State:        bp.State,
		Region:       bp.Region,
		Email:        bp.Email,
		PhoneNumber:  bp.PhoneNumber,
	}
}
