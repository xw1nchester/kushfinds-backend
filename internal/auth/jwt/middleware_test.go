package jwtauth

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	mockjwt "github.com/vetrovegor/kushfinds-backend/internal/auth/jwt/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestAuthMiddleware(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockTokenManager := mockjwt.NewMockJwtManager(ctrl) // предполагается, что мок сгенерирован
	logger := zap.NewNop()
	middleware := NewMiddleware(logger, mockTokenManager)

	tests := []struct {
		name               string
		authHeader         string
		setupMock          func()
		expectedStatusCode int
		expectedUserID     *int // nil если не должно попасть в next
	}{
		{
			name:               "No auth header",
			authHeader:         "",
			setupMock:          func() {},
			expectedStatusCode: http.StatusUnauthorized,
			expectedUserID:     nil,
		},
		{
			name:               "Invalid format",
			authHeader:         "Bearer",
			setupMock:          func() {},
			expectedStatusCode: http.StatusUnauthorized,
			expectedUserID:     nil,
		},
		{
			name:       "Invalid token",
			authHeader: "Bearer invalid.token.here",
			setupMock: func() {
				mockTokenManager.EXPECT().
					ParseToken("invalid.token.here").
					Return(0, errors.New("invalid token"))
			},
			expectedStatusCode: http.StatusUnauthorized,
			expectedUserID:     nil,
		},
		{
			name:       "Valid token",
			authHeader: "Bearer valid.token",
			setupMock: func() {
				mockTokenManager.EXPECT().
					ParseToken("valid.token").
					Return(42, nil)
			},
			expectedStatusCode: http.StatusOK,
			expectedUserID:     ptr(42),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupMock()

			req := httptest.NewRequest(http.MethodGet, "/some-protected-route", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}
			rec := httptest.NewRecorder()

			var actualUserID *int

			protectedHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if v := r.Context().Value(UserIDContextKey{}); v != nil {
					uid := v.(int)
					actualUserID = &uid
				}
				w.WriteHeader(http.StatusOK)
			})

			handlerToTest := middleware(protectedHandler)
			handlerToTest.ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)
			assert.Equal(t, tt.expectedUserID, actualUserID)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}
