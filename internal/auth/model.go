package auth

import "github.com/vetrovegor/kushfinds-backend/internal/user"

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type AuthResponse struct {
	User       user.User `json:"user"`
	AcessToken string    `json:"accessToken"`
}
