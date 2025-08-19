package jwtauth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xw1nchester/kushfinds-backend/internal/config"
)

type manager struct {
	jwtConfig config.JWT
}

func NewManager(jwtConfig config.JWT) *manager {
	return &manager{
		jwtConfig: jwtConfig,
	}
}

type UserClaims struct {
	UserID  int  `json:"user_id"`
	IsAdmin bool `json:"is_admin"`
}

type customClaims struct {
	jwt.RegisteredClaims
	UserClaims
}

func (m *manager) GenerateToken(user UserClaims) (string, error) {
	customClaims := customClaims{
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.jwtConfig.AccessTokenTTL)),
		},
		user,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)

	return token.SignedString([]byte(m.jwtConfig.Secret))
}

func (tm *manager) GetRefreshTokenTTL() time.Duration {
	return tm.jwtConfig.RefreshTokenTTL
}

func (tm *manager) ParseToken(tokenStr string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &customClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tm.jwtConfig.Secret), nil
	})
	if err != nil || !token.Valid {
		return nil, err
	}

	claims, ok := token.Claims.(*customClaims)
	if !ok {
		return nil, err
	}

	return &claims.UserClaims, nil
}
