package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	authDB "github.com/vetrovegor/kushfinds-backend/internal/auth/db"
	"github.com/vetrovegor/kushfinds-backend/internal/code"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	"github.com/vetrovegor/kushfinds-backend/pkg/transactor"
	"go.uber.org/zap"
)

var (
	ErrInvalidCredentials    = apperror.NewAppError("invalid credentials")
	ErrUserAlreadyVerified   = apperror.NewAppError("the user has already been verified")
	ErrInvalidCode           = apperror.NewAppError("invalid code")
	ErrEmailAlreadyExists    = apperror.NewAppError("the user with this email already exists")
	ErrCodeAlreadySent       = apperror.NewAppError("code has already been sent")
	ErrNicknameAlreadySet    = apperror.NewAppError("the nickname is already set")
	ErrPasswordAlreadySet    = apperror.NewAppError("the password is already set")
	ErrUsernameAlreadyExists = apperror.NewAppError("the user with this username already exists")
	ErrUserNotVerified       = apperror.NewAppError("the user has not been verified")
	ErrPasswordNotSet        = apperror.NewAppError("the user does not have a password set")
)

//go:generate mockgen -destination=mocks/repo/mock.go -package=mockauthrepo . Repository
type Repository interface {
	CreateSession(ctx context.Context, token string, userAgent string, userID int, expiryDate time.Time) error
	DeleteNotExpirySessionByToken(ctx context.Context, token string) (int, error)
}

//go:generate mockgen -destination=mocks/user/mock.go -package=mockuserservice . UserService
type UserService interface {
	GetByID(ctx context.Context, id int) (*user.User, error)
	GetByEmail(ctx context.Context, email string) (*user.User, error)
	Create(ctx context.Context, email string) (int, error)
	Verify(ctx context.Context, id int) (*user.User, error)
	CheckUsernameIsAvailable(ctx context.Context, username string) (bool, error)
	SetProfileInfo(ctx context.Context, data *user.User) (*user.User, error)
	SetPassword(ctx context.Context, id int, passwordHash []byte) error
}

//go:generate mockgen -destination=mocks/code/mock.go -package=mockcodeservice . CodeService
type CodeService interface {
	GenerateVerify(ctx context.Context, userID int) (string, error)
	ValidateVerify(ctx context.Context, code string, userID int) error
}

//go:generate mockgen -destination=mocks/mail/mock.go -package=mockmail . MailManager
type MailManager interface {
	SendMail(subject string, body string, to []string) error
}

//go:generate mockgen -destination=mocks/token/mock.go -package=mocktoken . TokenManager
type TokenManager interface {
	GenerateToken(userID int) (string, error)
	GetRefreshTokenTTL() time.Duration
}

//go:generate mockgen -destination=mocks/password/mock.go -package=mockpassword . PasswordManager
type PasswordManager interface {
	GenerateHashFromPassword(password []byte) ([]byte, error)
	CompareHashAndPassword(hashedPassword []byte, password []byte) error
}

// TODO: рефакторить
type service struct {
	authRepository  Repository
	userService     UserService
	codeService     CodeService
	tokenManager    TokenManager
	mailManager     MailManager
	passwordManager PasswordManager
	txManager       transactor.Manager
	logger          *zap.Logger
}

func NewService(
	authRepository Repository,
	userService UserService,
	codeService CodeService,
	tokenManager TokenManager,
	mailManager MailManager,
	passwordManager PasswordManager,
	txManager transactor.Manager,
	logger *zap.Logger,
) *service {
	return &service{
		authRepository:  authRepository,
		userService:     userService,
		codeService:     codeService,
		tokenManager:    tokenManager,
		mailManager:     mailManager,
		passwordManager: passwordManager,
		txManager:       txManager,
		logger:          logger,
	}
}

func (s *service) generateTokens(ctx context.Context, userAgent string, userID int) (*auth.Tokens, error) {
	accessToken, err := s.tokenManager.GenerateToken(userID)
	if err != nil {
		s.logger.Error("unexpected error when generating jwt token", zap.Error(err))

		return nil, err
	}

	refreshToken := uuid.New().String()
	expiryDate := time.Now().Add(s.tokenManager.GetRefreshTokenTTL())

	err = s.authRepository.CreateSession(ctx, refreshToken, userAgent, userID, expiryDate)
	if err != nil {
		s.logger.Error("unexpected error when generating refresh token", zap.Error(err))
		return nil, err
	}

	return &auth.Tokens{
		JwtToken:     auth.JwtToken{AccessToken: accessToken},
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) RegisterEmail(ctx context.Context, dto auth.EmailRequest) error {
	_, err := s.userService.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, apperror.ErrNotFound) {
		return err
	}

	if err == nil {
		return apperror.NewAppError("the user with this email already exists")
	}

	var generatedCode string

	err = s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		userID, err := s.userService.Create(ctx, dto.Email)
		if err != nil {
			return err
		}

		generatedCode, err = s.codeService.GenerateVerify(ctx, userID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	go func() {
		if err := s.mailManager.SendMail(
			"Confirmation of registration",
			fmt.Sprintf("Your registration confirmation code: %s", generatedCode),
			[]string{dto.Email},
		); err != nil {
			s.logger.Error("unexpected error when sending email", zap.Error(err))
		}
	}()

	return nil
}

func (s *service) RegisterVerify(ctx context.Context, dto auth.CodeRequest, userAgent string) (*auth.AuthFullResponse, error) {
	existingUser, err := s.userService.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	if existingUser.IsVerified {
		return nil, ErrUserAlreadyVerified
	}

	err = s.codeService.ValidateVerify(ctx, dto.Code, existingUser.ID)
	if err != nil {
		if errors.Is(err, code.ErrCodeNotFound) {
			return nil, ErrInvalidCode
		}

		return nil, err
	}

	var verifiedUser *user.User
	var tokens *auth.Tokens

	err = s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		verifiedUser, err = s.userService.Verify(ctx, existingUser.ID)
		if err != nil {
			return err
		}

		tokens, err = s.generateTokens(ctx, userAgent, verifiedUser.ID)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return &auth.AuthFullResponse{
		UserResponse: user.UserResponse{User: *verifiedUser},
		Tokens:       *tokens,
	}, nil
}

func (s *service) VerifyResend(ctx context.Context, dto auth.EmailRequest) error {
	existingUser, err := s.userService.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return ErrInvalidCredentials
		}

		return err
	}

	if existingUser.IsVerified {
		return ErrUserAlreadyVerified
	}

	generatedCode, err := s.codeService.GenerateVerify(ctx, existingUser.ID)
	if err != nil {
		if errors.Is(err, code.ErrCodeAlreadySent) {
			return ErrCodeAlreadySent
		}

		return err
	}

	go func() {
		if err := s.mailManager.SendMail(
			"Confirmation of registration",
			fmt.Sprintf("Your registration confirmation code: %s", generatedCode),
			[]string{dto.Email},
		); err != nil {
			s.logger.Error("unexpected error when sending email", zap.Error(err))
		}
	}()

	return nil
}

func (s *service) SaveProfileInfo(ctx context.Context, userID int, dto auth.ProfileRequest) (*user.UserResponse, error) {
	existingUser, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if existingUser.Username != nil {
		return nil, ErrNicknameAlreadySet
	}

	usernameIsAvailable, err := s.userService.CheckUsernameIsAvailable(ctx, dto.Username)
	if err != nil && !errors.Is(err, apperror.ErrNotFound) {
		return nil, err
	}

	if !usernameIsAvailable {
		return nil, ErrUsernameAlreadyExists
	}

	updatedUser, err := s.userService.SetProfileInfo(
		ctx,
		&user.User{
			ID:        userID,
			Username:  &dto.Username,
			FirstName: &dto.FirstName,
			LastName:  &dto.LastName,
		},
	)
	if err != nil {
		return nil, err
	}

	return &user.UserResponse{User: *updatedUser}, nil
}

func (s *service) SavePassword(ctx context.Context, userID int, dto auth.PasswordRequest) error {
	existingUser, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if existingUser.IsPasswordSet {
		return ErrPasswordAlreadySet
	}

	passHash, err := s.passwordManager.GenerateHashFromPassword([]byte(dto.Password))
	if err != nil {
		return err
	}

	if err = s.userService.SetPassword(ctx, userID, passHash); err != nil {
		return err
	}

	return nil
}

func (s *service) GetUserByEmail(ctx context.Context, dto auth.EmailRequest) (*user.UserResponse, error) {
	existingUser, err := s.userService.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	return &user.UserResponse{User: *existingUser}, nil
}

func (s *service) Login(ctx context.Context, dto auth.EmailPasswordRequest, userAgent string) (*auth.AuthFullResponse, error) {
	existingUser, err := s.userService.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	if !existingUser.IsVerified {
		return nil, ErrUserNotVerified
	}

	if !existingUser.IsPasswordSet {
		return nil, ErrPasswordNotSet
	}

	if err := s.passwordManager.CompareHashAndPassword(*existingUser.PasswordHash, []byte(dto.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	tokens, err := s.generateTokens(ctx, userAgent, existingUser.ID)
	if err != nil {
		return nil, err
	}

	return &auth.AuthFullResponse{
		UserResponse: user.UserResponse{User: *existingUser},
		Tokens:       *tokens,
	}, nil
}

func (s *service) Refresh(ctx context.Context, token string, userAgent string) (*auth.Tokens, error) {
	var tokens *auth.Tokens

	if err := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		userID, err := s.authRepository.DeleteNotExpirySessionByToken(ctx, token)
		if err != nil {
			if !errors.Is(err, authDB.ErrNotFound) {
				s.logger.Error("unexpected error when deleting refresh token", zap.Error(err))
			}
			return err
		}

		tokens, err = s.generateTokens(ctx, userAgent, userID)
		if err != nil {
			return err
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *service) Logout(ctx context.Context, token string) error {
	_, err := s.authRepository.DeleteNotExpirySessionByToken(ctx, token)
	if err != nil && !errors.Is(err, authDB.ErrNotFound) {
		s.logger.Error("unexpected error when deleting refresh token", zap.Error(err))
	}

	return err
}
