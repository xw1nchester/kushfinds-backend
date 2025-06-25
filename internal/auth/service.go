package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	authDB "github.com/vetrovegor/kushfinds-backend/internal/auth/db"
	jwtauth "github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	"github.com/vetrovegor/kushfinds-backend/internal/code"
	"github.com/vetrovegor/kushfinds-backend/pkg/transactor"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
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

//go:generate mockgen -source=service.go -destination=mocks/mock.go
type Service interface {
	RegisterEmail(ctx context.Context, dto EmailRequest) error
	RegisterVerify(ctx context.Context, dto CodeRequest, userAgent string) (*AuthFullResponse, error)
	VerifyResend(ctx context.Context, dto EmailRequest) error
	SaveProfileInfo(ctx context.Context, userID int, dto ProfileRequest) (*user.UserResponse, error)
	SavePassword(ctx context.Context, userID int, dto PasswordRequest) error
	GetUserByEmail(ctx context.Context, dto EmailRequest) (*user.UserResponse, error)
	Login(ctx context.Context, dto EmailPasswordRequest, userAgent string) (*AuthFullResponse, error)
	Refresh(ctx context.Context, token string, userAgent string) (*Tokens, error)
	Logout(ctx context.Context, token string) error
}

// TODO: рефакторить
type service struct {
	authRepository authDB.Repository
	userService    user.Service
	codeService    code.Service
	tokenManager   jwtauth.TokenManager
	mailManager    mailManager
	txManager      transactor.Manager
	logger         *zap.Logger
}

func NewService(
	authRepository authDB.Repository,
	userService user.Service,
	codeService code.Service,
	tokenManager jwtauth.TokenManager,
	mailManager mailManager,
	txManager transactor.Manager,
	logger *zap.Logger,
) Service {
	return &service{
		authRepository: authRepository,
		userService:    userService,
		codeService:    codeService,
		tokenManager:   tokenManager,
		mailManager:    mailManager,
		txManager:      txManager,
		logger:         logger,
	}
}

func (s *service) generateTokens(ctx context.Context, userAgent string, userID int) (*Tokens, error) {
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

	return &Tokens{
		JwtToken:     JwtToken{AccessToken: accessToken},
		RefreshToken: refreshToken,
	}, nil
}

func (s *service) RegisterEmail(ctx context.Context, dto EmailRequest) error {
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

func (s *service) RegisterVerify(ctx context.Context, dto CodeRequest, userAgent string) (*AuthFullResponse, error) {
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
	var tokens *Tokens

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

	return &AuthFullResponse{
		UserResponse: user.UserResponse{User: *verifiedUser},
		Tokens:       *tokens,
	}, nil
}

func (s *service) VerifyResend(ctx context.Context, dto EmailRequest) error {
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

func (s *service) SaveProfileInfo(ctx context.Context, userID int, dto ProfileRequest) (*user.UserResponse, error) {
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

func (s *service) SavePassword(ctx context.Context, userID int, dto PasswordRequest) error {
	existingUser, err := s.userService.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	if existingUser.PasswordHash != nil {
		return ErrPasswordAlreadySet
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("unexpected error when hashing password", zap.Error(err))
		return err
	}

	if err = s.userService.SetPassword(ctx, userID, passHash); err != nil {
		return err
	}

	return nil
}

func (s *service) GetUserByEmail(ctx context.Context, dto EmailRequest) (*user.UserResponse, error) {
	existingUser, err := s.userService.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, apperror.ErrNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	return &user.UserResponse{User: *existingUser}, nil
}

func (s *service) Login(ctx context.Context, dto EmailPasswordRequest, userAgent string) (*AuthFullResponse, error) {
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

	if existingUser.PasswordHash == nil {
		return nil, ErrPasswordNotSet
	}

	if err := bcrypt.CompareHashAndPassword(*existingUser.PasswordHash, []byte(dto.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	tokens, err := s.generateTokens(ctx, userAgent, existingUser.ID)
	if err != nil {
		return nil, err
	}

	return &AuthFullResponse{
		UserResponse: user.UserResponse{User: *existingUser},
		Tokens:       *tokens,
	}, nil
}

func (s *service) Refresh(ctx context.Context, token string, userAgent string) (*Tokens, error) {
	var tokens *Tokens
	
	err := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
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

		return  nil
	})

	if err != nil {
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
