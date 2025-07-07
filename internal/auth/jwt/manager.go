package jwtauth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
)

//go:generate mockgen -source=manager.go -destination=mocks/mock.go -package=mockjwt
type Manager interface {
	GenerateToken(userID int) (string, error)
	GetRefreshTokenTTL() time.Duration
	ParseToken(tokenStr string) (int, error)
}

type manager struct {
	jwtConfig config.JWT
}

func NewTokenManager(jwtConfig config.JWT) Manager {
	return &manager{
		jwtConfig: jwtConfig,
	}
}

type CustomClaims struct {
	jwt.RegisteredClaims
	UserID int `json:"user_id"`
}

func (m *manager) GenerateToken(userID int) (string, error) {
	customClaims := CustomClaims{
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(m.jwtConfig.AccessTokenTTL)),
		},
		userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)

	return token.SignedString([]byte(m.jwtConfig.Secret))
}

func (tm *manager) GetRefreshTokenTTL() time.Duration {
	return tm.jwtConfig.RefreshTokenTTL
}

func (tm *manager) ParseToken(tokenStr string) (int, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(tm.jwtConfig.Secret), nil
	})
	if err != nil || !token.Valid {
		return 0, err
	}

	claims, ok := token.Claims.(*CustomClaims)
	if !ok {
		return 0, err
	}

	return claims.UserID, nil
}
