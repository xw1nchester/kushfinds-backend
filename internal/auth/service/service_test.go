package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	mockauthdb "github.com/vetrovegor/kushfinds-backend/internal/auth/db/mocks"
	mockjwt "github.com/vetrovegor/kushfinds-backend/internal/auth/jwt/mocks"
	mockmail "github.com/vetrovegor/kushfinds-backend/internal/auth/mocks"
	"github.com/vetrovegor/kushfinds-backend/internal/code"
	mockcodeservice "github.com/vetrovegor/kushfinds-backend/internal/code/mocks"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	mockuserservice "github.com/vetrovegor/kushfinds-backend/internal/user/mocks"
	mocktransactor "github.com/vetrovegor/kushfinds-backend/pkg/transactor/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

const (
	Email           = "test@mail.ru"
	Code            = "12345"
	UserAgent       = "Go-http-client/1.1"
	AccessToken     = "some.access.token"
	RefreshToken    = "refresh-token"
	RefreshTokenTTL = 720 * time.Hour
)

func TestGenerateTokens(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockTokenManager *mockjwt.MockTokenManager,
		mockAuthRepo *mockauthdb.MockRepository,
		userAgent string,
		userID int,
	)

	tests := []struct {
		name                string
		userAgent           string
		userID              int
		mockBehavior        mockBehavior
		expectedError       error
		expectedAccessToken string
	}{
		{
			name:      "success",
			userAgent: UserAgent,
			userID:    1,
			mockBehavior: func(
				ctx context.Context,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				userAgent string,
				userID int,
			) {
				mockTokenManager.EXPECT().GenerateToken(userID).Return(AccessToken, nil)
				mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
				mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, userID, gomock.Any()).Return(nil)
			},
			expectedError:       nil,
			expectedAccessToken: AccessToken,
		},
		{
			name:      "access token generation error",
			userAgent: UserAgent,
			userID:    1,
			mockBehavior: func(
				ctx context.Context,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				userAgent string,
				userID int,
			) {
				mockTokenManager.EXPECT().GenerateToken(userID).Return("", errors.New("some error"))
			},
			expectedError:       errors.New("some error"),
			expectedAccessToken: "",
		},
		{
			name:      "creating session error",
			userAgent: UserAgent,
			userID:    1,
			mockBehavior: func(
				ctx context.Context,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				userAgent string,
				userID int,
			) {
				mockTokenManager.EXPECT().GenerateToken(userID).Return(AccessToken, nil)
				mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
				mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, userID, gomock.Any()).Return(errors.New("some error"))
			},
			expectedError:       errors.New("some error"),
			expectedAccessToken: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockTokenManager := mockjwt.NewMockTokenManager(ctrl)
			mockAuthRepo := mockauthdb.NewMockRepository(ctrl)

			service := &service{
				tokenManager:   mockTokenManager,
				authRepository: mockAuthRepo,
				logger:         zap.NewNop(),
			}

			ctx := context.Background()
			tt.mockBehavior(ctx, mockTokenManager, mockAuthRepo, tt.userAgent, tt.userID)

			resp, err := service.generateTokens(ctx, tt.userAgent, tt.userID)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tt.expectedAccessToken, resp.JwtToken.AccessToken)
				require.NotEmpty(t, resp.RefreshToken)
			}
		})
	}
}

func TestRegisterEmail(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockUserService *mockuserservice.MockService,
		mockTxManager *mocktransactor.MockManager,
		mockCodeService *mockcodeservice.MockService,
		mockMailManager *mockmail.MockMailManager,
		dto auth.EmailRequest,
	)

	tests := []struct {
		name          string
		dto           auth.EmailRequest
		mockBehavior  mockBehavior
		expectedError error
	}{
		{
			name: "successful registration",
			dto:  auth.EmailRequest{Email: Email},
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				userID := 1
				code := "123456"

				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, apperror.ErrNotFound)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(context.Context) error) error {
						mockUserService.EXPECT().Create(ctx, dto.Email).Return(userID, nil)
						mockCodeService.EXPECT().GenerateVerify(ctx, userID).Return(code, nil)
						return fn(ctx)
					},
				)
				mockMailManager.EXPECT().SendMail(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			expectedError: nil,
		},
		{
			name: "email already exists",
			dto:  auth.EmailRequest{Email: Email},
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(&user.User{ID: 2}, nil)
			},
			expectedError: apperror.NewAppError("the user with this email already exists"),
		},
		{
			name: "error in user creation inside transaction",
			dto:  auth.EmailRequest{Email: Email},
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, apperror.ErrNotFound)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(context.Context) error) error {
						mockUserService.EXPECT().Create(ctx, dto.Email).Return(0, errors.New("user creation error"))
						return fn(ctx)
					},
				)
			},
			expectedError: errors.New("user creation error"),
		},
		{
			name: "error in code creation inside transaction",
			dto:  auth.EmailRequest{Email: Email},
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				userID := 1

				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, apperror.ErrNotFound)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(context.Context) error) error {
						mockUserService.EXPECT().Create(ctx, dto.Email).Return(userID, nil)
						mockCodeService.EXPECT().GenerateVerify(ctx, userID).Return("", errors.New("code creation error"))
						return fn(ctx)
					},
				)
			},
			expectedError: errors.New("code creation error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()

			mockUserService := mockuserservice.NewMockService(ctrl)
			mockTxManager := mocktransactor.NewMockManager(ctrl)
			mockCodeService := mockcodeservice.NewMockService(ctrl)
			mockMailManager := mockmail.NewMockMailManager(ctrl)

			service := &service{
				userService: mockUserService,
				txManager:   mockTxManager,
				codeService: mockCodeService,
				mailManager: mockMailManager,
				logger:      zap.NewNop(),
			}

			tt.mockBehavior(ctx, mockUserService, mockTxManager, mockCodeService, mockMailManager, tt.dto)

			err := service.RegisterEmail(ctx, tt.dto)

			if tt.expectedError != nil {
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRegisterVerify(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockUserService *mockuserservice.MockService,
		mockCodeService *mockcodeservice.MockService,
		mockTxManager *mocktransactor.MockManager,
		mockTokenManager *mockjwt.MockTokenManager,
		mockAuthRepo *mockauthdb.MockRepository,
		dto auth.CodeRequest,
		userAgent string,
	)

	tests := []struct {
		name          string
		dto           auth.CodeRequest
		userAgent     string
		mockBehavior  mockBehavior
		expectedError error
		expectedUser  *user.User
	}{
		{
			name:      "success",
			dto:       auth.CodeRequest{Email: Email, Code: Code},
			userAgent: UserAgent,
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockCodeService *mockcodeservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				dto auth.CodeRequest,
				userAgent string,
			) {
				userID := 42
				existingUser := &user.User{ID: userID, Email: dto.Email, IsVerified: false}
				verifiedUser := &user.User{ID: userID, Email: dto.Email, IsVerified: true}
				accessToken := AccessToken

				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(existingUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, userID).Return(nil)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						mockUserService.EXPECT().Verify(ctx, userID).Return(verifiedUser, nil)
						mockTokenManager.EXPECT().GenerateToken(userID).Return(accessToken, nil)
						mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
						mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, userID, gomock.Any()).Return(nil)
						return fn(ctx)
					},
				)
			},
			expectedError: nil,
			expectedUser:  &user.User{ID: 42, Email: Email, IsVerified: true},
		},
		{
			name:      "user not found",
			dto:       auth.CodeRequest{Email: Email, Code: Code},
			userAgent: UserAgent,
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockCodeService *mockcodeservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				dto auth.CodeRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, apperror.ErrNotFound)
			},
			expectedError: ErrInvalidCredentials,
		},
		{
			name:      "user already verified",
			dto:       auth.CodeRequest{Email: Email, Code: Code},
			userAgent: UserAgent,
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockCodeService *mockcodeservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				dto auth.CodeRequest,
				userAgent string,
			) {
				user := &user.User{ID: 1, Email: dto.Email, IsVerified: true}
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(user, nil)
			},
			expectedError: ErrUserAlreadyVerified,
		},
		{
			name:      "invalid code",
			dto:       auth.CodeRequest{Email: Email, Code: Code},
			userAgent: UserAgent,
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockCodeService *mockcodeservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				dto auth.CodeRequest,
				userAgent string,
			) {
				userID := 10
				user := &user.User{ID: userID, Email: dto.Email, IsVerified: false}
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(user, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, userID).Return(code.ErrCodeNotFound)
			},
			expectedError: ErrInvalidCode,
		},
		{
			name:      "error in verify user",
			dto:       auth.CodeRequest{Email: Email, Code: Code},
			userAgent: UserAgent,
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockCodeService *mockcodeservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				dto auth.CodeRequest,
				userAgent string,
			) {
				userID := 5
				user := &user.User{ID: userID, Email: dto.Email, IsVerified: false}
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(user, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, userID).Return(nil)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						mockUserService.EXPECT().Verify(ctx, userID).Return(nil, errors.New("verify error"))
						return fn(ctx)
					},
				)
			},
			expectedError: errors.New("verify error"),
		},
		{
			name:      "error generating token",
			dto:       auth.CodeRequest{Email: Email, Code: Code},
			userAgent: UserAgent,
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockCodeService *mockcodeservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				dto auth.CodeRequest,
				userAgent string,
			) {
				userID := 5
				noVerifiedUser := &user.User{ID: userID, Email: dto.Email, IsVerified: false}
				verifiedUser := &user.User{ID: userID, Email: dto.Email, IsVerified: true}

				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(noVerifiedUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, userID).Return(nil)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						mockUserService.EXPECT().Verify(ctx, userID).Return(verifiedUser, nil)
						mockTokenManager.EXPECT().GenerateToken(userID).Return("", errors.New("token generation error"))
						return fn(ctx)
					},
				)
			},
			expectedError: errors.New("token generation error"),
		},
		{
			name:      "error creating session",
			dto:       auth.CodeRequest{Email: Email, Code: Code},
			userAgent: UserAgent,
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockCodeService *mockcodeservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				dto auth.CodeRequest,
				userAgent string,
			) {
				userID := 5
				noVerifiedUser := &user.User{ID: userID, Email: dto.Email, IsVerified: false}
				verifiedUser := &user.User{ID: userID, Email: dto.Email, IsVerified: true}

				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(noVerifiedUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, userID).Return(nil)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						mockUserService.EXPECT().Verify(ctx, userID).Return(verifiedUser, nil)
						mockTokenManager.EXPECT().GenerateToken(userID).Return("token", nil)
						mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
						mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, userID, gomock.Any()).Return(errors.New("session error"))
						return fn(ctx)
					},
				)
			},
			expectedError: errors.New("session error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserService := mockuserservice.NewMockService(ctrl)
			mockCodeService := mockcodeservice.NewMockService(ctrl)
			mockTxManager := mocktransactor.NewMockManager(ctrl)
			mockTokenManager := mockjwt.NewMockTokenManager(ctrl)
			mockAuthRepo := mockauthdb.NewMockRepository(ctrl)

			service := &service{
				userService:    mockUserService,
				codeService:    mockCodeService,
				txManager:      mockTxManager,
				tokenManager:   mockTokenManager,
				authRepository: mockAuthRepo,
				logger:         zap.NewNop(),
			}

			ctx := context.Background()
			tt.mockBehavior(ctx, mockUserService, mockCodeService, mockTxManager, mockTokenManager, mockAuthRepo, tt.dto, tt.userAgent)

			resp, err := service.RegisterVerify(ctx, tt.dto, tt.userAgent)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tt.expectedUser.ID, resp.UserResponse.User.ID)
				require.Equal(t, tt.expectedUser.Email, resp.UserResponse.User.Email)
			}
		})
	}
}
