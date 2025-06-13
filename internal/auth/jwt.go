package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
)

type tokenManager struct {
	jwtConfig config.JWT
}

func NewTokenManager(jwtConfig config.JWT) tokenManager {
	return tokenManager{
		jwtConfig: jwtConfig,
	}
}

type CustomClaims struct {
	jwt.RegisteredClaims
	UserID int `json:"user_id"`
}

func (tm tokenManager) GenerateToken(userID int) (string, error) {
	customClaims := CustomClaims{
		jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(tm.jwtConfig.AccessTokenTTL)),
		},
		userID,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, customClaims)

	return token.SignedString([]byte(tm.jwtConfig.Secret))
}

func (tm tokenManager) GetRefreshTokenTTL() time.Duration {
	return tm.jwtConfig.RefreshTokenTTL
}

func ParseToken(tokenStr string, secretKey string) (int, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &CustomClaims{}, func(token *jwt.Token) (any, error) {
		return []byte(secretKey), nil
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
