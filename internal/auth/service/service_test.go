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
	UserID          = 1
	Email           = "test@mail.ru"
	Code            = "12345"
	UserAgent       = "Go-http-client/1.1"
	AccessToken     = "some.access.token"
	RefreshTokenTTL = 720 * time.Hour
)

var (
	UnverifiedUser = &user.User{ID: UserID, Email: Email, IsVerified: false}
	VerifiedUser   = &user.User{ID: UserID, Email: Email, IsVerified: true}

	ErrUnexpected = errors.New("unexpected error")
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
		mockBehavior        mockBehavior
		expectedError       error
		expectedAccessToken string
	}{
		{
			name: "success",
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
			name: "access token generation error",
			mockBehavior: func(
				ctx context.Context,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				userAgent string,
				userID int,
			) {
				mockTokenManager.EXPECT().GenerateToken(userID).Return("", ErrUnexpected)
			},
			expectedError:       ErrUnexpected,
			expectedAccessToken: "",
		},
		{
			name: "creating session error",
			mockBehavior: func(
				ctx context.Context,
				mockTokenManager *mockjwt.MockTokenManager,
				mockAuthRepo *mockauthdb.MockRepository,
				userAgent string,
				userID int,
			) {
				mockTokenManager.EXPECT().GenerateToken(userID).Return(AccessToken, nil)
				mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
				mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, userID, gomock.Any()).Return(ErrUnexpected)
			},
			expectedError:       ErrUnexpected,
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
			tt.mockBehavior(ctx, mockTokenManager, mockAuthRepo, UserAgent, UserID)

			resp, err := service.generateTokens(ctx, UserAgent, UserID)

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
		mockBehavior  mockBehavior
		expectedError error
	}{
		{
			name: "successful registration",
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
						mockUserService.EXPECT().Create(ctx, dto.Email).Return(UserID, nil)
						mockCodeService.EXPECT().GenerateVerify(ctx, UserID).Return(Code, nil)
						return fn(ctx)
					},
				)
				mockMailManager.EXPECT().SendMail(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			expectedError: nil,
		},
		{
			name: "email already exists",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
			},
			expectedError: apperror.NewAppError("the user with this email already exists"),
		},
		{
			name: "unexpected error when fetching existing user",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "error in user creation inside transaction",
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

			tt.mockBehavior(ctx, mockUserService, mockTxManager, mockCodeService, mockMailManager, auth.EmailRequest{Email: Email})

			err := service.RegisterEmail(ctx, auth.EmailRequest{Email: Email})

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
		mockBehavior  mockBehavior
		expectedError error
		expectedUser  *user.User
	}{
		{
			name: "success",
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
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, UserID).Return(nil)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						mockUserService.EXPECT().Verify(ctx, UserID).Return(VerifiedUser, nil)
						mockTokenManager.EXPECT().GenerateToken(UserID).Return(AccessToken, nil)
						mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
						mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, UserID, gomock.Any()).Return(nil)
						return fn(ctx)
					},
				)
			},
			expectedError: nil,
			expectedUser:  VerifiedUser,
		},
		{
			name: "user not found",
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
			name: "unexpected error when fetching existing user",
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
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "user already verified",
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
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(VerifiedUser, nil)
			},
			expectedError: ErrUserAlreadyVerified,
		},
		{
			name: "invalid code",
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
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, UserID).Return(code.ErrCodeNotFound)
			},
			expectedError: ErrInvalidCode,
		},
		{
			name: "unexpected error when validating code",
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
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, UserID).Return(ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "error in verify user",
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
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, UserID).Return(nil)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						mockUserService.EXPECT().Verify(ctx, UserID).Return(nil, errors.New("verify error"))
						return fn(ctx)
					},
				)
			},
			expectedError: errors.New("verify error"),
		},
		{
			name: "error generating token",
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
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, UserID).Return(nil)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						mockUserService.EXPECT().Verify(ctx, UserID).Return(VerifiedUser, nil)
						mockTokenManager.EXPECT().GenerateToken(UserID).Return("", errors.New("token generation error"))
						return fn(ctx)
					},
				)
			},
			expectedError: errors.New("token generation error"),
		},
		{
			name: "error creating session",
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
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, UserID).Return(nil)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						mockUserService.EXPECT().Verify(ctx, UserID).Return(VerifiedUser, nil)
						mockTokenManager.EXPECT().GenerateToken(UserID).Return("token", nil)
						mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
						mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, UserID, gomock.Any()).Return(errors.New("session error"))
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
			tt.mockBehavior(
				ctx,
				mockUserService,
				mockCodeService,
				mockTxManager,
				mockTokenManager,
				mockAuthRepo,
				auth.CodeRequest{Email: Email, Code: Code},
				UserAgent,
			)

			resp, err := service.RegisterVerify(
				ctx, auth.CodeRequest{Email: Email, Code: Code},
				UserAgent,
			)

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
