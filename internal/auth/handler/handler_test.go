package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	mock_auth "github.com/vetrovegor/kushfinds-backend/internal/auth/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

func TestHandler_registerEmailHandler(t *testing.T) {
	type mockBehavior func(s *mock_auth.MockService, ctx context.Context, dto auth.EmailRequest)

	log := zap.NewNop()
	defer log.Sync()

	testTable := []struct {
		name               string
		inputBody          string
		inputDto           auth.EmailRequest
		mockBehavior       mockBehavior
		expectedStatusCode int
	}{
		{
			name:      "OK",
			inputBody: `{"email":"test@mail.ru"}`,
			inputDto:  auth.EmailRequest{Email: "test@mail.ru"},
			mockBehavior: func(s *mock_auth.MockService, ctx context.Context, dto auth.EmailRequest) {
				s.EXPECT().RegisterEmail(gomock.Any(), dto).Return(nil)
			},
			expectedStatusCode: 200,
		},
		{
			name:      "Invalid email",
			inputBody: `{"email":"invalid"}`,
			mockBehavior: func(s *mock_auth.MockService, ctx context.Context, dto auth.EmailRequest) {},
			expectedStatusCode: 400,
		},
		{
			name:      "Empty request body",
			inputBody: "{}",
			mockBehavior: func(s *mock_auth.MockService, ctx context.Context, dto auth.EmailRequest) {},
			expectedStatusCode: 400,
		},
		{
			name:      "Service decode body failure",
			inputBody: `{"email":"test@mail.ru"}`,
			inputDto:  auth.EmailRequest{Email: "test@mail.ru"},
			mockBehavior: func(s *mock_auth.MockService, ctx context.Context, dto auth.EmailRequest) {
				s.EXPECT().RegisterEmail(gomock.Any(), dto).Return(apperror.ErrDecodeBody)
			},
			expectedStatusCode: 400,
		},
		{
			name:      "Service unexpected failure",
			inputBody: `{"email":"test@mail.ru"}`,
			inputDto:  auth.EmailRequest{Email: "test@mail.ru"},
			mockBehavior: func(s *mock_auth.MockService, ctx context.Context, dto auth.EmailRequest) {
				s.EXPECT().RegisterEmail(gomock.Any(), dto).Return(errors.New("unexpected error"))
			},
			expectedStatusCode: 500,
		},
	}

	for _, tc := range testTable {
		t.Run(tc.name, func(t *testing.T) {
			c := gomock.NewController(t)
			defer c.Finish()

			authService := mock_auth.NewMockService(c)
			tc.mockBehavior(authService, context.Background(), tc.inputDto)

			handler := NewHandler(authService, authMiddleware, log)

			router := chi.NewRouter()

			router.Post("/register", apperror.Middleware(handler.RegisterEmailHandler))

			w := httptest.NewRecorder()
			req := httptest.NewRequest(
				http.MethodPost,
				"/register", 
				bytes.NewBufferString(tc.inputBody),
			)

			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatusCode, w.Code)
		})
	}
}

func authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}
