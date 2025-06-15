package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authDB "github.com/vetrovegor/kushfinds-backend/internal/auth/db"
	"github.com/vetrovegor/kushfinds-backend/internal/code"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	userDB "github.com/vetrovegor/kushfinds-backend/internal/user/db"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInternal              = errors.New("internal error")
	ErrEmailAlreadyExists    = errors.New("the user with this email already exists")
	ErrUserAlreadyVerified   = errors.New("the user has already been verified")
	ErrInvalidCredentials    = errors.New("invalid credentials")
	ErrInvalidCode           = errors.New("invalid code")
	ErrCodeAlreadySent       = errors.New("code has already been sent")
	ErrNicknameAlreadySet    = errors.New("the nickname is already set")
	ErrPasswordAlreadySet    = errors.New("the password is already set")
	ErrUsernameAlreadyExists = errors.New("the user with this username already exists")
	ErrUserNotVerified       = errors.New("the user has not been verified")
)

// TODO: разделять
type Service interface {
	RegisterEmail(ctx context.Context, dto EmailRequest) error
	RegisterVerify(ctx context.Context, dto CodeRequest, userAgent string) (*AuthFullResponse, error)
	VerifyResend(ctx context.Context, dto EmailRequest) error
	SaveProfileInfo(ctx context.Context, userID int, dto ProfileRequest) (*UserResponse, error)
	SavePassword(ctx context.Context, userID int, dto PasswordRequest) error
	GetUserByEmail(ctx context.Context, dto EmailRequest) (*UserResponse, error)
	Login(ctx context.Context, dto EmailPasswordRequest, userAgent string) (*AuthFullResponse, error)
	Refresh(ctx context.Context, token string, userAgent string) (*Tokens, error)
	Logout(ctx context.Context, token string) error
}

// TODO: рефакторить
type service struct {
	userRepository userDB.Repository
	authRepository authDB.Repository
	codeService    code.Service
	tokenManager   tokenManager
	mailManager    mailManager
	logger         *zap.Logger
}

func NewService(
	userRepository userDB.Repository,
	authRepository authDB.Repository,
	codeService code.Service,
	tokenManager tokenManager,
	mailManager mailManager,
	logger *zap.Logger,
) Service {
	return &service{
		userRepository: userRepository,
		authRepository: authRepository,
		codeService:    codeService,
		tokenManager:   tokenManager,
		mailManager:    mailManager,
		logger:         logger,
	}
}

func (s service) generateTokens(ctx context.Context, userAgent string, userID int) (*Tokens, error) {
	accessToken, err := s.tokenManager.GenerateToken(userID)
	if err != nil {
		s.logger.Error("error when generating jwt token", zap.Error(err))

		return nil, ErrInternal
	}

	refreshToken := uuid.New().String()
	expiryDate := time.Now().Add(s.tokenManager.GetRefreshTokenTTL())

	err = s.authRepository.CreateSession(ctx, refreshToken, userAgent, userID, expiryDate)
	if err != nil {
		s.logger.Error("error when generating refresh token", zap.Error(err))
		return nil, ErrInternal
	}

	return &Tokens{
		JwtToken:     JwtToken{AccessToken: accessToken},
		RefreshToken: refreshToken,
	}, nil
}

// TODO: нужна транзакция
func (s service) RegisterEmail(ctx context.Context, dto EmailRequest) error {
	_, err := s.userRepository.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, userDB.ErrUserNotFound) {
		s.logger.Error("error when fetching user", zap.Error(err))
		return ErrInternal
	}

	if err == nil {
		return ErrEmailAlreadyExists
	}

	userID, err := s.userRepository.Create(ctx, dto.Email)
	if err != nil {
		s.logger.Error("error when creating user", zap.Error(err))
		return ErrInternal
	}

	generatedCode, err := s.codeService.GenerateVerify(ctx, userID)
	if err != nil {
		return ErrInternal
	}

	go func() {
		if err := s.mailManager.SendMail(
			"Confirmation of registration",
			fmt.Sprintf("Your registration confirmation code: %s", generatedCode),
			[]string{dto.Email},
		); err != nil {
			s.logger.Error("error when sending email", zap.Error(err))
		}
	}()

	return nil
}

func (s service) RegisterVerify(ctx context.Context, dto CodeRequest, userAgent string) (*AuthFullResponse, error) {
	existingUser, err := s.userRepository.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, userDB.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		s.logger.Error("error when fetching user", zap.Error(err))
		return nil, ErrInternal
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

	verifiedUser, err := s.userRepository.Verify(ctx, existingUser.ID)
	if err != nil {
		s.logger.Error("error when verifying user", zap.Error(err))
		return nil, ErrInternal
	}

	tokens, err := s.generateTokens(ctx, userAgent, verifiedUser.ID)
	if err != nil {
		return nil, ErrInternal
	}

	return &AuthFullResponse{
		UserResponse: UserResponse{User: *verifiedUser},
		Tokens:       *tokens,
	}, nil
}

func (s service) VerifyResend(ctx context.Context, dto EmailRequest) error {
	existingUser, err := s.userRepository.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, userDB.ErrUserNotFound) {
			return ErrInvalidCredentials
		}

		s.logger.Error("error when fetching user", zap.Error(err))
		return ErrInternal
	}

	if existingUser.IsVerified {
		return ErrUserAlreadyVerified
	}

	generatedCode, err := s.codeService.GenerateVerify(ctx, existingUser.ID)
	if err != nil {
		if errors.Is(err, code.ErrCodeAlreadySent) {
			return ErrCodeAlreadySent
		}

		return ErrInternal
	}

	go func() {
		if err := s.mailManager.SendMail(
			"Confirmation of registration",
			fmt.Sprintf("Your registration confirmation code: %s", generatedCode),
			[]string{dto.Email},
		); err != nil {
			s.logger.Error("error when sending email", zap.Error(err))
		}
	}()

	return nil
}

// SaveProfileInfo implements Service.
func (s service) SaveProfileInfo(ctx context.Context, userID int, dto ProfileRequest) (*UserResponse, error) {
	existingUser, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userDB.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		s.logger.Error("error when fetching user", zap.Error(err))
		return nil, ErrInternal
	}

	if existingUser.Username != nil {
		return nil, ErrNicknameAlreadySet
	}

	usernameIsAvailable, err := s.userRepository.CheckUsernameIsAvailable(ctx, dto.Username)
	if err != nil && !errors.Is(err, userDB.ErrUserNotFound) {
		s.logger.Error("error when fetching user by username", zap.Error(err))

		return nil, ErrInternal
	}

	if !usernameIsAvailable {
		return nil, ErrUsernameAlreadyExists
	}

	updatedUser, err := s.userRepository.SetProfileInfo(
		ctx,
		user.User{
			ID:        userID,
			Username:  &dto.Username,
			FirstName: &dto.FirstName,
			LastName:  &dto.LastName,
		},
	)
	if err != nil {
		s.logger.Error("error when updating user profile", zap.Error(err))
		return nil, ErrInternal
	}

	return &UserResponse{User: *updatedUser}, nil
}

func (s service) SavePassword(ctx context.Context, userID int, dto PasswordRequest) error {
	existingUser, err := s.userRepository.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, userDB.ErrUserNotFound) {
			return ErrInvalidCredentials
		}

		s.logger.Error("error when fetching user", zap.Error(err))
		return ErrInternal
	}

	if existingUser.PasswordHash != nil {
		return ErrPasswordAlreadySet
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("error when hashing password", zap.Error(err))
		return ErrInternal
	}

	err = s.userRepository.SetPassword(ctx, userID, passHash)
	if err != nil {
		s.logger.Error("error when set user password", zap.Error(err))
		return ErrInternal
	}

	return nil
}

func (s service) GetUserByEmail(ctx context.Context, dto EmailRequest) (*UserResponse, error) {
	existingUser, err := s.userRepository.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, userDB.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		s.logger.Error("error when fetching user", zap.Error(err))
		return nil, ErrInternal
	}

	return &UserResponse{User: *existingUser}, nil
}

func (s service) Login(ctx context.Context, dto EmailPasswordRequest, userAgent string) (*AuthFullResponse, error) {
	existingUser, err := s.userRepository.GetByEmail(ctx, dto.Email)
	if err != nil {
		if errors.Is(err, userDB.ErrUserNotFound) {
			return nil, ErrInvalidCredentials
		}

		s.logger.Error("error when fetching user", zap.Error(err))
		return nil, ErrInternal
	}

	if !existingUser.IsVerified {
		return nil, ErrUserNotVerified
	}

	// TODO: подумать как обработать если у пользователя не установлен пароль

	if err := bcrypt.CompareHashAndPassword(*existingUser.PasswordHash, []byte(dto.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	tokens, err := s.generateTokens(ctx, userAgent, existingUser.ID)
	if err != nil {
		return nil, ErrInternal
	}

	return &AuthFullResponse{
		UserResponse: UserResponse{User: *existingUser},
		Tokens:       *tokens,
	}, nil
}

func (s service) Refresh(ctx context.Context, token string, userAgent string) (*Tokens, error) {
	userID, err := s.authRepository.DeleteNotExpirySessionByToken(ctx, token)
	if err != nil {
		if !errors.Is(err, authDB.ErrNotFound) {
			s.logger.Error("error when deleting refresh token", zap.Error(err))
		}
		return nil, err
	}

	tokens, err := s.generateTokens(ctx, userAgent, userID)
	if err != nil {
		return nil, err
	}

	return tokens, nil
}

func (s *service) Logout(ctx context.Context, token string) error {
	_, err := s.authRepository.DeleteNotExpirySessionByToken(ctx, token)
	if err != nil && !errors.Is(err, authDB.ErrNotFound) {
		s.logger.Error("error when deleting refresh token", zap.Error(err))
	}

	return err
}
