package auth

import "github.com/vetrovegor/kushfinds-backend/internal/user"

type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type AuthResponse struct {
	User       user.User `json:"user"`
	AcessToken string    `json:"accessToken"`
}
