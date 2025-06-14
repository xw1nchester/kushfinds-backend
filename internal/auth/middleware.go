package auth

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

func NewAuthMiddleware(logger *zap.Logger, secretKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

			userID, err := ParseToken(tokenStr, secretKey)
			if err != nil {
				logger.Warn("error when parsing JWT token", zap.Error(err))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			// TODO: сделать константу = "user_id" или тип того
			ctx := context.WithValue(r.Context(), "user_id", userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

