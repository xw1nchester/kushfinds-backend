package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	jwtauth "github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	mock_auth "github.com/vetrovegor/kushfinds-backend/internal/auth/mocks"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	"go.uber.org/mock/gomock"
)

func TestRegisterEmailHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_auth.NewMockService(ctrl)

	type fields struct {
		service auth.Service
	}

	type args struct {
		body      interface{}
		mockSetup func()
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
	}{
		{
			name:   "Success case",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "test@example.com",
				},
				mockSetup: func() {
					mockService.EXPECT().
						RegisterEmail(gomock.Any(), auth.EmailRequest{
							Email: "test@example.com",
						}).
						Return(nil)
				},
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:   "Invalid JSON body",
			fields: fields{service: mockService},
			args: args{
				body: "invalid json",
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error (empty email)",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error (invalid email format)",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "not-an-email",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Service returns known error (NotFound)",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "test@example.com",
				},
				mockSetup: func() {
					mockService.EXPECT().
						RegisterEmail(gomock.Any(), auth.EmailRequest{
							Email: "test@example.com",
						}).
						Return(apperror.ErrNotFound)
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:   "Service returns unknown error (500)",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "test@example.com",
				},
				mockSetup: func() {
					mockService.EXPECT().
						RegisterEmail(gomock.Any(), auth.EmailRequest{
							Email: "test@example.com",
						}).
						Return(errors.New("internal error"))
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup()
			}

			h := &handler{
				service: tt.fields.service,
			}

			var bodyReader *bytes.Reader
			switch v := tt.args.body.(type) {
			case string:
				bodyReader = bytes.NewReader([]byte(v))
			default:
				bodyBytes, err := json.Marshal(v)
				require.NoError(t, err)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/register/email", bodyReader)
			rec := httptest.NewRecorder()

			apperror.Middleware(h.RegisterEmailHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)
		})
	}
}

func TestRegisterVerifyHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_auth.NewMockService(ctrl)

	type fields struct {
		service auth.Service
	}

	type args struct {
		body      interface{}
		userAgent string
		mockSetup func()
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
		checkResponse      func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "Success case",
			fields: fields{service: mockService},
			args: args{
				body: auth.CodeRequest{
					Email: "test@example.com",
					Code:  "123456",
				},
				userAgent: "Go-http-client/1.1",
				mockSetup: func() {
					mockService.EXPECT().
						RegisterVerify(gomock.Any(), auth.CodeRequest{
							Email: "test@example.com",
							Code:  "123456",
						}, "Go-http-client/1.1").
						Return(&auth.AuthFullResponse{
							UserResponse: user.UserResponse{
								User: user.User{
									ID:         1,
									Email:      "test@example.com",
									IsVerified: true,
								},
							},
							Tokens: auth.Tokens{
								JwtToken: auth.JwtToken{
									AccessToken: "mockedAccessToken",
								},
								RefreshToken: "mockedRefreshToken",
							},
						}, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				res := rec.Result()
				defer res.Body.Close()

				cookies := res.Cookies()
				require.NotEmpty(t, cookies)
				assert.Equal(t, RefreshTokenCookieName, cookies[0].Name)
				assert.Equal(t, "mockedRefreshToken", cookies[0].Value)

				var responseBody auth.AuthResponse
				err := json.NewDecoder(res.Body).Decode(&responseBody)
				require.NoError(t, err)
				assert.Equal(t, 1, responseBody.User.ID)
				assert.Equal(t, "mockedAccessToken", responseBody.AccessToken)
			},
		},
		{
			name:   "Invalid JSON body",
			fields: fields{service: mockService},
			args: args{
				body:      "invalid json",
				userAgent: "Go-http-client/1.1",
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error (empty fields)",
			fields: fields{service: mockService},
			args: args{
				body:      auth.CodeRequest{},
				userAgent: "Go-http-client/1.1",
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Service returns known AppError (NotFound)",
			fields: fields{service: mockService},
			args: args{
				body: auth.CodeRequest{
					Email: "notfound@example.com",
					Code:  "123456",
				},
				userAgent: "Go-http-client/1.1",
				mockSetup: func() {
					mockService.EXPECT().
						RegisterVerify(gomock.Any(), auth.CodeRequest{
							Email: "notfound@example.com",
							Code:  "123456",
						}, "Go-http-client/1.1").
						Return(nil, apperror.ErrNotFound)
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:   "Service returns unknown error (500)",
			fields: fields{service: mockService},
			args: args{
				body: auth.CodeRequest{
					Email: "fail@example.com",
					Code:  "123456",
				},
				userAgent: "Go-http-client/1.1",
				mockSetup: func() {
					mockService.EXPECT().
						RegisterVerify(gomock.Any(), auth.CodeRequest{
							Email: "fail@example.com",
							Code:  "123456",
						}, "Go-http-client/1.1").
						Return(nil, errors.New("unexpected error"))
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup()
			}

			h := &handler{
				service: tt.fields.service,
			}

			var bodyReader *bytes.Reader
			switch v := tt.args.body.(type) {
			case string:
				bodyReader = bytes.NewReader([]byte(v))
			default:
				bodyBytes, err := json.Marshal(v)
				require.NoError(t, err)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/register/verify", bodyReader)
			req.Header.Set("User-Agent", tt.args.userAgent)
			rec := httptest.NewRecorder()

			apperror.Middleware(h.registerVerifyHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestVerifyResendHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_auth.NewMockService(ctrl)

	type args struct {
		body      interface{}
		mockSetup func()
	}

	tests := []struct {
		name               string
		args               args
		expectedStatusCode int
	}{
		{
			name: "Success case",
			args: args{
				body: auth.EmailRequest{
					Email: "test@example.com",
				},
				mockSetup: func() {
					mockService.EXPECT().
						VerifyResend(gomock.Any(), auth.EmailRequest{
							Email: "test@example.com",
						}).
						Return(nil)
				},
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name: "Invalid JSON body",
			args: args{
				body: "invalid json",
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "Validation error (empty email)",
			args: args{
				body: auth.EmailRequest{},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name: "Service returns NotFound error",
			args: args{
				body: auth.EmailRequest{
					Email: "notfound@example.com",
				},
				mockSetup: func() {
					mockService.EXPECT().
						VerifyResend(gomock.Any(), auth.EmailRequest{
							Email: "notfound@example.com",
						}).
						Return(apperror.ErrNotFound)
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name: "Service returns unknown error",
			args: args{
				body: auth.EmailRequest{
					Email: "test@example.com",
				},
				mockSetup: func() {
					mockService.EXPECT().
						VerifyResend(gomock.Any(), auth.EmailRequest{
							Email: "test@example.com",
						}).
						Return(errors.New("unexpected error"))
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup()
			}

			h := &handler{
				service: mockService,
			}

			var bodyReader *bytes.Reader
			switch v := tt.args.body.(type) {
			case string:
				bodyReader = bytes.NewReader([]byte(v))
			default:
				bodyBytes, err := json.Marshal(v)
				require.NoError(t, err)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/verify/resend", bodyReader)
			rec := httptest.NewRecorder()

			apperror.Middleware(h.VerifyResendHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)
		})
	}
}

func TestRegisterProfileHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_auth.NewMockService(ctrl)

	type fields struct {
		service auth.Service
	}

	type args struct {
		body      interface{}
		userID    interface{}
		mockSetup func()
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
	}{
		{
			name:   "Success case",
			fields: fields{service: mockService},
			args: args{
				body: auth.ProfileRequest{
					Username:  "testuser",
					FirstName: "Test",
					LastName:  "User",
				},
				userID: 1,
				mockSetup: func() {
					mockService.EXPECT().
						SaveProfileInfo(gomock.Any(), 1, auth.ProfileRequest{
							Username:  "testuser",
							FirstName: "Test",
							LastName:  "User",
						}).
						Return(&user.UserResponse{
							User: user.User{
								ID:        1,
								Email:     "test@example.com",
								Username:  ptrStr("testuser"),
								FirstName: ptrStr("Test"),
								LastName:  ptrStr("User"),
							},
						}, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:   "Invalid JSON",
			fields: fields{service: mockService},
			args: args{
				body:   "invalid json",
				userID: 1,
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error (empty fields)",
			fields: fields{service: mockService},
			args: args{
				body:   auth.ProfileRequest{},
				userID: 1,
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Service returns known error (NotFound)",
			fields: fields{service: mockService},
			args: args{
				body: auth.ProfileRequest{
					Username:  "testuser",
					FirstName: "Test",
					LastName:  "User",
				},
				userID: 1,
				mockSetup: func() {
					mockService.EXPECT().
						SaveProfileInfo(gomock.Any(), 1, gomock.Any()).
						Return(nil, apperror.ErrNotFound)
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:   "Service returns unknown error (500)",
			fields: fields{service: mockService},
			args: args{
				body: auth.ProfileRequest{
					Username:  "testuser",
					FirstName: "Test",
					LastName:  "User",
				},
				userID: 1,
				mockSetup: func() {
					mockService.EXPECT().
						SaveProfileInfo(gomock.Any(), 1, gomock.Any()).
						Return(nil, errors.New("internal error"))
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup()
			}

			h := &handler{
				service: tt.fields.service,
			}

			var bodyReader *bytes.Reader
			switch v := tt.args.body.(type) {
			case string:
				bodyReader = bytes.NewReader([]byte(v))
			default:
				bodyBytes, err := json.Marshal(v)
				require.NoError(t, err)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/register/profile", bodyReader)

			ctx := req.Context()
			if tt.args.userID != nil {
				ctx = context.WithValue(ctx, jwtauth.UserIDContextKey{}, tt.args.userID)
			}
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			apperror.Middleware(h.registerProfileHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)
		})
	}
}

func TestRegisterPasswordHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_auth.NewMockService(ctrl)

	type fields struct {
		service auth.Service
	}

	type args struct {
		body      interface{}
		mockSetup func()
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
	}{
		{
			name:   "Success case",
			fields: fields{service: mockService},
			args: args{
				body: auth.PasswordRequest{
					Password: "strongpassword",
				},
				mockSetup: func() {
					mockService.EXPECT().
						SavePassword(gomock.Any(), 1, auth.PasswordRequest{
							Password: "strongpassword",
						}).
						Return(nil)
				},
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:   "Invalid JSON body",
			fields: fields{service: mockService},
			args: args{
				body: "invalid json",
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error (empty password)",
			fields: fields{service: mockService},
			args: args{
				body: auth.PasswordRequest{
					Password: "",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error (too short password)",
			fields: fields{service: mockService},
			args: args{
				body: auth.PasswordRequest{
					Password: "short",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Service returns known error (Unauthorized)",
			fields: fields{service: mockService},
			args: args{
				body: auth.PasswordRequest{
					Password: "strongpassword",
				},
				mockSetup: func() {
					mockService.EXPECT().
						SavePassword(gomock.Any(), 1, auth.PasswordRequest{
							Password: "strongpassword",
						}).
						Return(apperror.ErrUnauthorized)
				},
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:   "Service returns unknown error (500)",
			fields: fields{service: mockService},
			args: args{
				body: auth.PasswordRequest{
					Password: "strongpassword",
				},
				mockSetup: func() {
					mockService.EXPECT().
						SavePassword(gomock.Any(), 1, auth.PasswordRequest{
							Password: "strongpassword",
						}).
						Return(errors.New("internal error"))
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup()
			}

			h := &handler{
				service: tt.fields.service,
			}

			var bodyReader *bytes.Reader
			switch v := tt.args.body.(type) {
			case string:
				bodyReader = bytes.NewReader([]byte(v))
			default:
				bodyBytes, err := json.Marshal(v)
				require.NoError(t, err)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/register/password", bodyReader)

			ctx := context.WithValue(req.Context(), jwtauth.UserIDContextKey{}, 1)
			req = req.WithContext(ctx)

			rec := httptest.NewRecorder()

			apperror.Middleware(h.registerPasswordHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)
		})
	}
}

func TestLoginEmailHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_auth.NewMockService(ctrl)

	type fields struct {
		service auth.Service
	}

	type args struct {
		body      interface{}
		mockSetup func()
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
	}{
		{
			name:   "Success case",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "test@example.com",
				},
				mockSetup: func() {
					mockService.EXPECT().
						GetUserByEmail(gomock.Any(), auth.EmailRequest{
							Email: "test@example.com",
						}).
						Return(&user.UserResponse{
							User: user.User{
								ID:    1,
								Email: "test@example.com",
							},
						}, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
		},
		{
			name:   "Invalid JSON body",
			fields: fields{service: mockService},
			args: args{
				body: "invalid json",
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error (empty email)",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error (invalid email format)",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "invalid-email",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Service returns not found error",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "notfound@example.com",
				},
				mockSetup: func() {
					mockService.EXPECT().
						GetUserByEmail(gomock.Any(), auth.EmailRequest{
							Email: "notfound@example.com",
						}).
						Return(nil, apperror.ErrNotFound)
				},
			},
			expectedStatusCode: http.StatusNotFound,
		},
		{
			name:   "Service returns unknown error (500)",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailRequest{
					Email: "error@example.com",
				},
				mockSetup: func() {
					mockService.EXPECT().
						GetUserByEmail(gomock.Any(), auth.EmailRequest{
							Email: "error@example.com",
						}).
						Return(nil, errors.New("unexpected error"))
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup()
			}

			h := &handler{
				service: tt.fields.service,
			}

			var bodyReader *bytes.Reader
			switch v := tt.args.body.(type) {
			case string:
				bodyReader = bytes.NewReader([]byte(v))
			default:
				bodyBytes, err := json.Marshal(v)
				require.NoError(t, err)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/login/email", bodyReader)
			rec := httptest.NewRecorder()

			apperror.Middleware(h.loginEmailHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)
		})
	}
}

func TestLoginPasswordHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_auth.NewMockService(ctrl)

	type fields struct {
		service auth.Service
	}

	type args struct {
		body      interface{}
		userAgent string
		mockSetup func()
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
		checkResponse      func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "Success case",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailPasswordRequest{
					Email:    "test@example.com",
					Password: "password123",
				},
				userAgent: "Go-http-client/1.1",
				mockSetup: func() {
					mockService.EXPECT().
						Login(gomock.Any(), auth.EmailPasswordRequest{
							Email:    "test@example.com",
							Password: "password123",
						}, "Go-http-client/1.1").
						Return(&auth.AuthFullResponse{
							UserResponse: user.UserResponse{
								User: user.User{
									ID:    1,
									Email: "test@example.com",
								},
							},
							Tokens: auth.Tokens{
								JwtToken: auth.JwtToken{
									AccessToken: "mockedAccessToken",
								},
								RefreshToken: "mockedRefreshToken",
							},
						}, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				res := rec.Result()
				defer res.Body.Close()

				cookies := res.Cookies()
				require.NotEmpty(t, cookies)
				assert.Equal(t, RefreshTokenCookieName, cookies[0].Name)
				assert.Equal(t, "mockedRefreshToken", cookies[0].Value)

				var responseBody auth.AuthResponse
				err := json.NewDecoder(res.Body).Decode(&responseBody)
				require.NoError(t, err)
				assert.Equal(t, 1, responseBody.User.ID)
				assert.Equal(t, "mockedAccessToken", responseBody.AccessToken)
			},
		},
		{
			name:   "Invalid JSON body",
			fields: fields{service: mockService},
			args: args{
				body:      "invalid json",
				userAgent: "Go-http-client/1.1",
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error: empty fields",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailPasswordRequest{},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error: invalid email",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailPasswordRequest{
					Email:    "invalid-email",
					Password: "password123",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Validation error: short password",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailPasswordRequest{
					Email:    "test@example.com",
					Password: "short",
				},
			},
			expectedStatusCode: http.StatusBadRequest,
		},
		{
			name:   "Service returns unauthorized",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailPasswordRequest{
					Email:    "unauth@example.com",
					Password: "password123",
				},
				userAgent: "Go-http-client/1.1",
				mockSetup: func() {
					mockService.EXPECT().
						Login(gomock.Any(), auth.EmailPasswordRequest{
							Email:    "unauth@example.com",
							Password: "password123",
						}, "Go-http-client/1.1").
						Return(nil, apperror.ErrUnauthorized)
				},
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:   "Service returns unknown error",
			fields: fields{service: mockService},
			args: args{
				body: auth.EmailPasswordRequest{
					Email:    "fail@example.com",
					Password: "password123",
				},
				userAgent: "Go-http-client/1.1",
				mockSetup: func() {
					mockService.EXPECT().
						Login(gomock.Any(), auth.EmailPasswordRequest{
							Email:    "fail@example.com",
							Password: "password123",
						}, "Go-http-client/1.1").
						Return(nil, errors.New("unexpected error"))
				},
			},
			expectedStatusCode: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup()
			}

			h := &handler{
				service: tt.fields.service,
			}

			var bodyReader *bytes.Reader
			switch v := tt.args.body.(type) {
			case string:
				bodyReader = bytes.NewReader([]byte(v))
			default:
				bodyBytes, err := json.Marshal(v)
				require.NoError(t, err)
				bodyReader = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/login/password", bodyReader)
			req.Header.Set("User-Agent", tt.args.userAgent)

			rec := httptest.NewRecorder()

			apperror.Middleware(h.loginPasswordHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestRefreshHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_auth.NewMockService(ctrl)

	type fields struct {
		service auth.Service
	}

	type args struct {
		cookieValue string
		userAgent   string
		mockSetup   func()
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
		checkResponse      func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "Success case",
			fields: fields{service: mockService},
			args: args{
				cookieValue: "valid_refresh_token",
				userAgent:   "Go-http-client/1.1",
				mockSetup: func() {
					mockService.EXPECT().
						Refresh(gomock.Any(), "valid_refresh_token", "Go-http-client/1.1").
						Return(&auth.Tokens{
							JwtToken: auth.JwtToken{
								AccessToken: "newAccessToken",
							},
							RefreshToken: "newRefreshToken",
						}, nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				res := rec.Result()
				defer res.Body.Close()

				cookies := res.Cookies()
				require.NotEmpty(t, cookies)
				assert.Equal(t, RefreshTokenCookieName, cookies[0].Name)
				assert.Equal(t, "newRefreshToken", cookies[0].Value)

				var responseBody auth.JwtToken
				err := json.NewDecoder(res.Body).Decode(&responseBody)
				require.NoError(t, err)
				assert.Equal(t, "newAccessToken", responseBody.AccessToken)
			},
		},
		{
			name:   "Missing cookie",
			fields: fields{service: mockService},
			args: args{
				userAgent: "Go-http-client/1.1",
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
		{
			name:   "Service returns error",
			fields: fields{service: mockService},
			args: args{
				cookieValue: "invalid_refresh_token",
				userAgent:   "Go-http-client/1.1",
				mockSetup: func() {
					mockService.EXPECT().
						Refresh(gomock.Any(), "invalid_refresh_token", "Go-http-client/1.1").
						Return(nil, errors.New("invalid token"))
				},
			},
			expectedStatusCode: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup()
			}

			h := &handler{
				service: tt.fields.service,
			}

			req := httptest.NewRequest(http.MethodPost, "/refresh", nil)
			req.Header.Set("User-Agent", tt.args.userAgent)

			if tt.args.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  RefreshTokenCookieName,
					Value: tt.args.cookieValue,
				})
			}

			rec := httptest.NewRecorder()

			apperror.Middleware(h.refreshHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockService := mock_auth.NewMockService(ctrl)

	type fields struct {
		service auth.Service
	}

	type args struct {
		cookieValue string
		mockSetup   func()
	}

	tests := []struct {
		name               string
		fields             fields
		args               args
		expectedStatusCode int
		checkResponse      func(t *testing.T, rec *httptest.ResponseRecorder)
	}{
		{
			name:   "Cookie present - success logout and clear cookie",
			fields: fields{service: mockService},
			args: args{
				cookieValue: "valid_refresh_token",
				mockSetup: func() {
					mockService.EXPECT().
						Logout(gomock.Any(), "valid_refresh_token").
						Return(nil)
				},
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				cookies := rec.Result().Cookies()
				require.NotEmpty(t, cookies)
				assert.Equal(t, RefreshTokenCookieName, cookies[0].Name)
				assert.Equal(t, "", cookies[0].Value)
				assert.Equal(t, -1, cookies[0].MaxAge)
			},
		},
		{
			name:   "Cookie absent - no call to service, no cookie set",
			fields: fields{service: mockService},
			args: args{
				mockSetup: func() {},
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				cookies := rec.Result().Cookies()
				assert.Empty(t, cookies)
			},
		},
		{
			name:   "Cookie present - service returns error (ignored)",
			fields: fields{service: mockService},
			args: args{
				cookieValue: "bad_token",
				mockSetup: func() {
					mockService.EXPECT().
						Logout(gomock.Any(), "bad_token").
						Return(errors.New("logout error"))
				},
			},
			expectedStatusCode: http.StatusOK,
			checkResponse: func(t *testing.T, rec *httptest.ResponseRecorder) {
				cookies := rec.Result().Cookies()
				require.NotEmpty(t, cookies)
				assert.Equal(t, "", cookies[0].Value)
				assert.Equal(t, -1, cookies[0].MaxAge)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.args.mockSetup != nil {
				tt.args.mockSetup()
			}

			h := &handler{
				service: tt.fields.service,
			}

			req := httptest.NewRequest(http.MethodPost, "/logout", nil)

			if tt.args.cookieValue != "" {
				req.AddCookie(&http.Cookie{
					Name:  RefreshTokenCookieName,
					Value: tt.args.cookieValue,
				})
			}

			rec := httptest.NewRecorder()

			apperror.Middleware(h.logoutHandler).ServeHTTP(rec, req)

			assert.Equal(t, tt.expectedStatusCode, rec.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rec)
			}
		})
	}
}


func ptrStr(s string) *string {
	return &s
}
