package auth

import "github.com/vetrovegor/kushfinds-backend/internal/user"

type EmailRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type CodeRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code" validate:"required"`
}

type JwtToken struct {
	AccessToken string `json:"accessToken"`
}

type Tokens struct {
	JwtToken
	RefreshToken string
}

type UserResponse struct {
	// TODO: стоит создать отдельный User в котором будет не вся инфа
	// в user.User убрать json теги
	User user.User `json:"user"`
}

type AuthFullResponse struct {
	UserResponse
	Tokens
}

type AuthResponse struct {
	UserResponse
	JwtToken
}

type ProfileRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=30"`
	FirstName string `json:"firstName" validate:"required,min=1,max=50"`
	LastName  string `json:"lastName" validate:"required,min=1,max=50"`
}

type PasswordRequest struct {
	Password string `json:"password" validate:"required,min=8"`
}

type EmailPasswordRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}
