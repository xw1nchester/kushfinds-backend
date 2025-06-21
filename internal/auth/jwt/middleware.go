package jwtauth

import (
	"context"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type UserIDContextKey struct{}

func NewAuthMiddleware(logger *zap.Logger, tokenManager TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			headerParts := strings.Split(authHeader, " ")
			if len(headerParts) != 2 || headerParts[0] != "Bearer" || headerParts[1] == "" {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			userID, err := tokenManager.ParseToken(headerParts[1])
			if err != nil {
				logger.Warn("error when parsing JWT token", zap.Error(err))
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDContextKey{}, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
