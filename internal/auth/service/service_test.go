package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	mockcodeservice "github.com/vetrovegor/kushfinds-backend/internal/auth/service/mocks/code"
	mockmail "github.com/vetrovegor/kushfinds-backend/internal/auth/service/mocks/mail"
	mockpassword "github.com/vetrovegor/kushfinds-backend/internal/auth/service/mocks/password"
	mockauthrepo "github.com/vetrovegor/kushfinds-backend/internal/auth/service/mocks/repo"
	mocktoken "github.com/vetrovegor/kushfinds-backend/internal/auth/service/mocks/token"
	mockuserservice "github.com/vetrovegor/kushfinds-backend/internal/auth/service/mocks/user"
	"github.com/vetrovegor/kushfinds-backend/internal/code"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	mocktransactor "github.com/vetrovegor/kushfinds-backend/pkg/transactor/mocks"
	"go.uber.org/mock/gomock"
	"go.uber.org/zap"
)

const (
	UserID          = 1
	Email           = "test@mail.ru"
	Username        = "l4ndar"
	FirstName       = "John"
	LastName        = "Doe"
	Password        = "qwertyuiop"
	Code            = "12345"
	UserAgent       = "Go-http-client/1.1"
	AccessToken     = "some.access.token"
	RefreshTokenTTL = 720 * time.Hour
)

var (
	// TODO: подумать как лучше
	PasswordHash = &[]byte{1}

	UnverifiedUser              = &user.User{ID: UserID, Email: Email, IsVerified: false}
	VerifiedUser                = &user.User{ID: UserID, Email: Email, IsVerified: true}
	VerifiedUserWithProfileInfo = &user.User{
		ID:            UserID,
		Email:         Email,
		IsVerified:    true,
		Username:      ptrStr(Username),
		FirstName:     ptrStr(FirstName),
		LastName:      ptrStr(LastName),
		IsPasswordSet: false,
		PasswordHash:  nil,
	}
	VerifiedUserWithProfileInfoAndPassword = &user.User{
		ID:            UserID,
		Email:         Email,
		IsVerified:    true,
		Username:      ptrStr(Username),
		FirstName:     ptrStr(FirstName),
		LastName:      ptrStr(LastName),
		IsPasswordSet: true,
		PasswordHash:  PasswordHash,
	}

	ErrUnexpected = errors.New("unexpected error")
)

func TestGenerateTokens(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockTokenManager *mocktoken.MockTokenManager,
		mockAuthRepo *mockauthrepo.MockRepository,
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
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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

			mockTokenManager := mocktoken.NewMockTokenManager(ctrl)
			mockAuthRepo := mockauthrepo.NewMockRepository(ctrl)

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
		mockUserService *mockuserservice.MockUserService,
		mockTxManager *mocktransactor.MockManager,
		mockCodeService *mockcodeservice.MockCodeService,
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
				mockUserService *mockuserservice.MockUserService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockCodeService,
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
				mockUserService *mockuserservice.MockUserService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockCodeService,
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
				mockUserService *mockuserservice.MockUserService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockCodeService,
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
				mockUserService *mockuserservice.MockUserService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockCodeService,
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
				mockUserService *mockuserservice.MockUserService,
				mockTxManager *mocktransactor.MockManager,
				mockCodeService *mockcodeservice.MockCodeService,
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

			mockUserService := mockuserservice.NewMockUserService(ctrl)
			mockTxManager := mocktransactor.NewMockManager(ctrl)
			mockCodeService := mockcodeservice.NewMockCodeService(ctrl)
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
		mockUserService *mockuserservice.MockUserService,
		mockCodeService *mockcodeservice.MockCodeService,
		mockTxManager *mocktransactor.MockManager,
		mockTokenManager *mocktoken.MockTokenManager,
		mockAuthRepo *mockauthrepo.MockRepository,
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
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
			name: "error when generating token",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				dto auth.CodeRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().ValidateVerify(ctx, dto.Code, UserID).Return(nil)
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(ctx context.Context) error) error {
						mockUserService.EXPECT().Verify(ctx, UserID).Return(VerifiedUser, nil)
						mockTokenManager.EXPECT().GenerateToken(UserID).Return("", ErrUnexpected)
						return fn(ctx)
					},
				)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "error when creating session",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockTxManager *mocktransactor.MockManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
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
						mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, UserID, gomock.Any()).Return(ErrUnexpected)
						return fn(ctx)
					},
				)
			},
			expectedError: ErrUnexpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserService := mockuserservice.NewMockUserService(ctrl)
			mockCodeService := mockcodeservice.NewMockCodeService(ctrl)
			mockTxManager := mocktransactor.NewMockManager(ctrl)
			mockTokenManager := mocktoken.NewMockTokenManager(ctrl)
			mockAuthRepo := mockauthrepo.NewMockRepository(ctrl)

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

func TestVerifyResend(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockUserService *mockuserservice.MockUserService,
		mockCodeService *mockcodeservice.MockCodeService,
		mockMailManager *mockmail.MockMailManager,
		dto auth.EmailRequest,
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
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().GenerateVerify(ctx, UnverifiedUser.ID).Return(Code, nil)
				mockMailManager.EXPECT().SendMail(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil).AnyTimes()
			},
			expectedError: nil,
		},
		{
			name: "user not found",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, apperror.ErrNotFound)
			},
			expectedError: ErrInvalidCredentials,
		},
		{
			name: "db error when fetching user",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "already verified user",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(VerifiedUser, nil)
			},
			expectedError: ErrUserAlreadyVerified,
		},
		{
			name: "code already been sent",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().GenerateVerify(ctx, UnverifiedUser.ID).Return("", code.ErrCodeAlreadySent)
			},
			expectedError: ErrCodeAlreadySent,
		},
		{
			name: "db error when save code",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockCodeService *mockcodeservice.MockCodeService,
				mockMailManager *mockmail.MockMailManager,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(UnverifiedUser, nil)
				mockCodeService.EXPECT().GenerateVerify(ctx, UnverifiedUser.ID).Return("", ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserService := mockuserservice.NewMockUserService(ctrl)
			mockCodeService := mockcodeservice.NewMockCodeService(ctrl)
			mockMailManager := mockmail.NewMockMailManager(ctrl)

			service := &service{
				userService: mockUserService,
				codeService: mockCodeService,
				mailManager: mockMailManager,
				logger:      zap.NewNop(),
			}

			ctx := context.Background()
			tt.mockBehavior(
				ctx,
				mockUserService,
				mockCodeService,
				mockMailManager,
				auth.EmailRequest{Email: Email},
			)

			err := service.VerifyResend(
				ctx, auth.EmailRequest{Email: Email},
			)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestSaveProfileInfo(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockUserService *mockuserservice.MockUserService,
		userID int,
		dto auth.ProfileRequest,
	)

	tests := []struct {
		name          string
		mockBehavior  mockBehavior
		expectedError error
		expectedResp  *user.UserResponse
	}{
		{
			name: "success",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				userID int,
				dto auth.ProfileRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(VerifiedUser, nil)
				mockUserService.EXPECT().CheckUsernameIsAvailable(ctx, dto.Username).Return(true, nil)
				mockUserService.EXPECT().SetProfileInfo(
					ctx,
					&user.User{
						ID:        userID,
						Username:  &dto.Username,
						FirstName: &dto.FirstName,
						LastName:  &dto.LastName,
					},
				).Return(VerifiedUserWithProfileInfo, nil)
			},
			expectedError: nil,
			expectedResp:  &user.UserResponse{User: *VerifiedUserWithProfileInfo},
		},
		{
			name: "db error when fetching user",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				userID int,
				dto auth.ProfileRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(nil, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
			expectedResp:  nil,
		},
		{
			name: "username already set",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				userID int,
				dto auth.ProfileRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(VerifiedUserWithProfileInfo, nil)
			},
			expectedError: ErrNicknameAlreadySet,
			expectedResp:  nil,
		},
		{
			name: "db error when check username availability",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				userID int,
				dto auth.ProfileRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(VerifiedUser, nil)
				mockUserService.EXPECT().CheckUsernameIsAvailable(ctx, dto.Username).Return(false, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
			expectedResp:  nil,
		},
		{
			name: "username is not available",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				userID int,
				dto auth.ProfileRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(VerifiedUser, nil)
				mockUserService.EXPECT().CheckUsernameIsAvailable(ctx, dto.Username).Return(false, nil)
			},
			expectedError: ErrUsernameAlreadyExists,
			expectedResp:  nil,
		},
		{
			name: "db error when set profile info",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				userID int,
				dto auth.ProfileRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(VerifiedUser, nil)
				mockUserService.EXPECT().CheckUsernameIsAvailable(ctx, dto.Username).Return(true, nil)
				mockUserService.EXPECT().SetProfileInfo(
					ctx,
					&user.User{
						ID:        userID,
						Username:  &dto.Username,
						FirstName: &dto.FirstName,
						LastName:  &dto.LastName,
					},
				).Return(nil, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
			expectedResp:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserService := mockuserservice.NewMockUserService(ctrl)

			service := &service{
				userService: mockUserService,
				logger:      zap.NewNop(),
			}

			ctx := context.Background()
			tt.mockBehavior(
				ctx,
				mockUserService,
				UserID,
				auth.ProfileRequest{
					Username:  Username,
					FirstName: FirstName,
					LastName:  LastName,
				},
			)

			resp, err := service.SaveProfileInfo(
				ctx,
				UserID,
				auth.ProfileRequest{
					Username:  Username,
					FirstName: FirstName,
					LastName:  LastName,
				},
			)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tt.expectedResp.User.ID, resp.User.ID)
				require.Equal(t, *tt.expectedResp.User.Username, *resp.User.Username)
				require.Equal(t, *tt.expectedResp.User.FirstName, *resp.User.FirstName)
				require.Equal(t, *tt.expectedResp.User.LastName, *resp.User.LastName)
			}
		})
	}
}

func TestSavePassword(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockUserService *mockuserservice.MockUserService,
		mockPasswordManager *mockpassword.MockPasswordManager,
		userID int,
		dto auth.PasswordRequest,
	)

	tests := []struct {
		name          string
		mockBehavior  mockBehavior
		expectedError error
	}{
		{
			name: "success",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				userID int,
				dto auth.PasswordRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(VerifiedUserWithProfileInfo, nil)
				mockPasswordManager.EXPECT().GenerateHashFromPassword([]byte(dto.Password)).Return(*PasswordHash, nil)
				mockUserService.EXPECT().SetPassword(ctx, userID, *PasswordHash).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "error when fetching user",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				userID int,
				dto auth.PasswordRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(nil, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "password is already set",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				userID int,
				dto auth.PasswordRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(VerifiedUserWithProfileInfoAndPassword, nil)
			},
			expectedError: ErrPasswordAlreadySet,
		},
		{
			name: "error when hashing password",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				userID int,
				dto auth.PasswordRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(VerifiedUserWithProfileInfo, nil)
				mockPasswordManager.EXPECT().GenerateHashFromPassword([]byte(dto.Password)).Return(nil, ErrUnexpected)

			},
			expectedError: ErrUnexpected,
		},
		{
			name: "error when saving password",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				userID int,
				dto auth.PasswordRequest,
			) {
				mockUserService.EXPECT().GetByID(ctx, userID).Return(VerifiedUserWithProfileInfo, nil)
				mockPasswordManager.EXPECT().GenerateHashFromPassword([]byte(dto.Password)).Return(*PasswordHash, nil)
				mockUserService.EXPECT().SetPassword(ctx, userID, *PasswordHash).Return(ErrUnexpected)

			},
			expectedError: ErrUnexpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserService := mockuserservice.NewMockUserService(ctrl)
			mockPasswordManager := mockpassword.NewMockPasswordManager(ctrl)

			service := &service{
				userService:     mockUserService,
				passwordManager: mockPasswordManager,
				logger:          zap.NewNop(),
			}

			ctx := context.Background()
			tt.mockBehavior(
				ctx,
				mockUserService,
				mockPasswordManager,
				UserID,
				auth.PasswordRequest{
					Password: Password,
				},
			)

			err := service.SavePassword(
				ctx,
				UserID,
				auth.PasswordRequest{
					Password: Password,
				},
			)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestGetUserByEmail(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockUserService *mockuserservice.MockUserService,
		dto auth.EmailRequest,
	)

	tests := []struct {
		name          string
		mockBehavior  mockBehavior
		expectedError error
		expectedResp  *user.UserResponse
	}{
		{
			name: "success",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(VerifiedUser, nil)
			},
			expectedError: nil,
			expectedResp:  &user.UserResponse{User: *VerifiedUser},
		},
		{
			name: "user not found",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, apperror.ErrNotFound)
			},
			expectedError: ErrInvalidCredentials,
			expectedResp:  nil,
		},
		{
			name: "db error when fetching user",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				dto auth.EmailRequest,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).Return(nil, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
			expectedResp:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserService := mockuserservice.NewMockUserService(ctrl)

			service := &service{
				userService: mockUserService,
				logger:      zap.NewNop(),
			}

			ctx := context.Background()
			tt.mockBehavior(
				ctx,
				mockUserService,
				auth.EmailRequest{
					Email: Email,
				},
			)

			resp, err := service.GetUserByEmail(
				ctx,
				auth.EmailRequest{
					Email: Email,
				},
			)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tt.expectedResp.User.Email, resp.User.Email)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockUserService *mockuserservice.MockUserService,
		mockPasswordManager *mockpassword.MockPasswordManager,
		mockTokenManager *mocktoken.MockTokenManager,
		mockAuthRepo *mockauthrepo.MockRepository,
		dto auth.EmailPasswordRequest,
		userAgent string,
	)

	tests := []struct {
		name          string
		mockBehavior  mockBehavior
		expectedError error
		expectedResp  *auth.AuthFullResponse
	}{
		{
			name: "success",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				dto auth.EmailPasswordRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).
					Return(VerifiedUserWithProfileInfoAndPassword, nil)
				mockPasswordManager.EXPECT().
					CompareHashAndPassword(*VerifiedUserWithProfileInfoAndPassword.PasswordHash, []byte(dto.Password)).
					Return(nil)
				mockTokenManager.EXPECT().GenerateToken(UserID).Return(AccessToken, nil)
				mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
				mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), UserAgent, UserID, gomock.Any())
			},
			expectedError: nil,
			expectedResp: &auth.AuthFullResponse{
				UserResponse: user.UserResponse{User: *VerifiedUserWithProfileInfoAndPassword},
				Tokens: auth.Tokens{
					JwtToken: auth.JwtToken{AccessToken: AccessToken},
				},
			},
		},
		{
			name: "user not found",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				tokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				dto auth.EmailPasswordRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).
					Return(nil, apperror.ErrNotFound)
			},
			expectedError: ErrInvalidCredentials,
		},
		{
			name: "unexpected error when fetching user",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				tokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				dto auth.EmailPasswordRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).
					Return(nil, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "user is not verified",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				tokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				dto auth.EmailPasswordRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).
					Return(UnverifiedUser, nil)
			},
			expectedError: ErrUserNotVerified,
		},
		{
			name: "password is not set",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				tokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				dto auth.EmailPasswordRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).
					Return(VerifiedUser, nil)
			},
			expectedError: ErrPasswordNotSet,
		},
		{
			name: "error when compare password",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				tokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				dto auth.EmailPasswordRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).
					Return(VerifiedUserWithProfileInfoAndPassword, nil)
				mockPasswordManager.EXPECT().
					CompareHashAndPassword(*VerifiedUserWithProfileInfoAndPassword.PasswordHash, []byte(dto.Password)).
					Return(ErrUnexpected)
			},
			expectedError: ErrInvalidCredentials,
		},
		{
			name: "error when generating token",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				dto auth.EmailPasswordRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).
					Return(VerifiedUserWithProfileInfoAndPassword, nil)
				mockPasswordManager.EXPECT().
					CompareHashAndPassword(*VerifiedUserWithProfileInfoAndPassword.PasswordHash, []byte(dto.Password)).
					Return(nil)
				mockTokenManager.EXPECT().GenerateToken(UserID).Return("", ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "error when creating session",
			mockBehavior: func(
				ctx context.Context,
				mockUserService *mockuserservice.MockUserService,
				mockPasswordManager *mockpassword.MockPasswordManager,
				mockTokenManager *mocktoken.MockTokenManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				dto auth.EmailPasswordRequest,
				userAgent string,
			) {
				mockUserService.EXPECT().GetByEmail(ctx, dto.Email).
					Return(VerifiedUserWithProfileInfoAndPassword, nil)
				mockPasswordManager.EXPECT().
					CompareHashAndPassword(*VerifiedUserWithProfileInfoAndPassword.PasswordHash, []byte(dto.Password)).
					Return(nil)
				mockTokenManager.EXPECT().GenerateToken(UserID).Return(AccessToken, nil)
				mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
				mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, UserID, gomock.Any()).Return(ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockUserService := mockuserservice.NewMockUserService(ctrl)
			mockTokenManager := mocktoken.NewMockTokenManager(ctrl)
			mockPasswordManager := mockpassword.NewMockPasswordManager(ctrl)
			mockAuthRepo := mockauthrepo.NewMockRepository(ctrl)

			service := &service{
				userService:     mockUserService,
				tokenManager:    mockTokenManager,
				passwordManager: mockPasswordManager,
				authRepository:  mockAuthRepo,
				logger:          zap.NewNop(),
			}

			ctx := context.Background()
			tt.mockBehavior(
				ctx,
				mockUserService,
				mockPasswordManager,
				mockTokenManager,
				mockAuthRepo,
				auth.EmailPasswordRequest{
					Email:    Email,
					Password: Password,
				},
				UserAgent,
			)

			resp, err := service.Login(
				ctx,
				auth.EmailPasswordRequest{
					Email:    Email,
					Password: Password,
				},
				UserAgent,
			)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tt.expectedResp.User.ID, resp.User.ID)
				require.Equal(t, tt.expectedResp.AccessToken, resp.AccessToken)
				require.NotEmpty(t, resp.RefreshToken)
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockTxManager *mocktransactor.MockManager,
		mockAuthRepo *mockauthrepo.MockRepository,
		mockTokenManager *mocktoken.MockTokenManager,
		token string,
		userAgent string,
	)

	tests := []struct {
		name          string
		mockBehavior  mockBehavior
		expectedError error
		expectedResp  *auth.Tokens
		token         string
		userAgent     string
	}{
		{
			name: "success",
			mockBehavior: func(
				ctx context.Context,
				mockTxManager *mocktransactor.MockManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				mockTokenManager *mocktoken.MockTokenManager,
				token string,
				userAgent string,
			) {
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(context.Context) error) error {
						mockAuthRepo.EXPECT().DeleteNotExpirySessionByToken(ctx, gomock.Any()).Return(UserID, nil)
						mockTokenManager.EXPECT().GenerateToken(UserID).Return(AccessToken, nil)
						mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
						mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), userAgent, UserID, gomock.Any())
						return fn(ctx)
					},
				)
			},
			expectedError: nil,
			expectedResp: &auth.Tokens{
				JwtToken: auth.JwtToken{AccessToken: AccessToken},
			},
		},
		{
			name: "error when deleting session",
			mockBehavior: func(
				ctx context.Context,
				mockTxManager *mocktransactor.MockManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				mockTokenManager *mocktoken.MockTokenManager,
				token string,
				userAgent string,
			) {
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(context.Context) error) error {
						mockAuthRepo.EXPECT().DeleteNotExpirySessionByToken(ctx, gomock.Any()).Return(0, ErrUnexpected)
						return fn(ctx)
					},
				)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "error when generating token",
			mockBehavior: func(
				ctx context.Context,
				mockTxManager *mocktransactor.MockManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				mockTokenManager *mocktoken.MockTokenManager,
				token string,
				userAgent string,
			) {
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(context.Context) error) error {
						mockAuthRepo.EXPECT().DeleteNotExpirySessionByToken(ctx, gomock.Any()).Return(UserID, nil)
						mockTokenManager.EXPECT().GenerateToken(UserID).Return("", ErrUnexpected)
						return fn(ctx)
					},
				)
			},
			expectedError: ErrUnexpected,
		},
		{
			name: "error when creating session",
			mockBehavior: func(
				ctx context.Context,
				mockTxManager *mocktransactor.MockManager,
				mockAuthRepo *mockauthrepo.MockRepository,
				mockTokenManager *mocktoken.MockTokenManager,
				token string,
				userAgent string,
			) {
				mockTxManager.EXPECT().WithinTransaction(ctx, gomock.Any()).DoAndReturn(
					func(ctx context.Context, fn func(context.Context) error) error {
						mockAuthRepo.EXPECT().DeleteNotExpirySessionByToken(ctx, gomock.Any()).Return(UserID, nil)
						mockTokenManager.EXPECT().GenerateToken(UserID).Return(AccessToken, nil)
						mockTokenManager.EXPECT().GetRefreshTokenTTL().Return(RefreshTokenTTL)
						mockAuthRepo.EXPECT().CreateSession(ctx, gomock.Any(), UserAgent, UserID, gomock.Any()).Return(ErrUnexpected)
						return fn(ctx)
					},
				)
			},
			expectedError: ErrUnexpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			ctx := context.Background()

			mockTxManager := mocktransactor.NewMockManager(ctrl)
			mockAuthRepo := mockauthrepo.NewMockRepository(ctrl)
			mockTokenManager := mocktoken.NewMockTokenManager(ctrl)

			service := &service{
				txManager:      mockTxManager,
				authRepository: mockAuthRepo,
				tokenManager:   mockTokenManager,
				logger:         zap.NewNop(),
			}

			tt.mockBehavior(ctx, mockTxManager, mockAuthRepo, mockTokenManager, gomock.Any().String(), UserAgent)

			resp, err := service.Refresh(ctx, gomock.Any().String(), UserAgent)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Equal(t, tt.expectedResp.AccessToken, resp.AccessToken)
				require.NotEmpty(t, resp.RefreshToken)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	type mockBehavior func(
		ctx context.Context,
		mockAuthRepo *mockauthrepo.MockRepository,
	)

	tests := []struct {
		name          string
		mockBehavior  mockBehavior
		expectedError error
	}{
		{
			name: "success",
			mockBehavior: func(
				ctx context.Context,
				mockAuthRepo *mockauthrepo.MockRepository,
			) {
				mockAuthRepo.EXPECT().DeleteNotExpirySessionByToken(ctx, gomock.Any()).Return(UserID, nil)
			},
			expectedError: nil,
		},
		{
			name: "success",
			mockBehavior: func(
				ctx context.Context,
				mockAuthRepo *mockauthrepo.MockRepository,
			) {
				mockAuthRepo.EXPECT().DeleteNotExpirySessionByToken(ctx, gomock.Any()).Return(UserID, ErrUnexpected)
			},
			expectedError: ErrUnexpected,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockAuthRepo := mockauthrepo.NewMockRepository(ctrl)

			service := &service{
				authRepository: mockAuthRepo,
				logger:         zap.NewNop(),
			}

			ctx := context.Background()
			tt.mockBehavior(
				ctx,
				mockAuthRepo,
			)

			err := service.Logout(
				ctx,
				gomock.Any().String(),
			)

			if tt.expectedError != nil {
				require.Error(t, err)
				require.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func ptrStr(s string) *string {
	return &s
}
