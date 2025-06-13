package auth

import "github.com/vetrovegor/kushfinds-backend/internal/user"

type RegisterEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type JwtToken struct {
	AccessToken  string `json:"accessToken"`
}

type Tokens struct {
	JwtToken
	RefreshToken string
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type AuthResponse struct {
	User       user.User `json:"user"`
	AcessToken string    `json:"accessToken"`
}
