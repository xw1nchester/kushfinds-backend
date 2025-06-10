package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
)

func GenerateToken(jwtConfig config.JWT, userID int) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"exp":     time.Now().Add(jwtConfig.TokenTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	return token.SignedString([]byte(jwtConfig.Secret))
}
