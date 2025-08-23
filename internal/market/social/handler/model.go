package socialhandler

import "github.com/xw1nchester/kushfinds-backend/internal/market/social"

type SocialsResponse struct {
	Socials []social.Social `json:"socials"`
}
