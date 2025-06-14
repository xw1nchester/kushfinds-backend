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

type Service interface {
	RegisterEmail(ctx context.Context, dto EmailRequest) error
	RegisterVerify(ctx context.Context, dto CodeRequest, userAgent string) (*AuthFullResponse, error)
	VerifyResend(ctx context.Context, dto EmailRequest) error
	SaveProfileInfo(ctx context.Context, userID int, dto ProfileRequest) (*user.User, error)
	SavePassword(ctx context.Context, userID int, dto PasswordRequest) error
	GetUserByEmail(ctx context.Context, dto EmailRequest) (*UserResponse, error)
	Login(ctx context.Context, dto EmailPasswordRequest, userAgent string) (*AuthFullResponse, error)
}

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

func (s service) generateRefreshToken(ctx context.Context, userAgent string, userID int) (string, error) {
	token := uuid.New().String()
	expiryDate := time.Now().Add(s.tokenManager.GetRefreshTokenTTL())

	err := s.authRepository.CreateSession(ctx, token, userAgent, userID, expiryDate)

	return token, err
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

	accessToken, err := s.tokenManager.GenerateToken(verifiedUser.ID)
	if err != nil {
		s.logger.Error("error when generating jwt token", zap.Error(err))

		return nil, ErrInternal
	}

	refreshToken, err := s.generateRefreshToken(ctx, userAgent, verifiedUser.ID)
	if err != nil {
		s.logger.Error("error when generating jwt token", zap.Error(err))
		return nil, ErrInternal
	}

	return &AuthFullResponse{
		UserResponse: UserResponse{User: *verifiedUser},
		Tokens: Tokens{
			JwtToken:     JwtToken{AccessToken: accessToken},
			RefreshToken: refreshToken,
		},
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
func (s service) SaveProfileInfo(ctx context.Context, userID int, dto ProfileRequest) (*user.User, error) {
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

	return updatedUser, nil
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

	accessToken, err := s.tokenManager.GenerateToken(existingUser.ID)
	if err != nil {
		s.logger.Error("error when generating jwt token", zap.Error(err))

		return nil, ErrInternal
	}

	refreshToken, err := s.generateRefreshToken(ctx, userAgent, existingUser.ID)
	if err != nil {
		s.logger.Error("error when generating jwt token", zap.Error(err))
		return nil, ErrInternal
	}

	return &AuthFullResponse{
		UserResponse: UserResponse{User: *existingUser},
		Tokens: Tokens{
			JwtToken:     JwtToken{AccessToken: accessToken},
			RefreshToken: refreshToken,
		},
	}, nil
}

// func (s Service) Login(ctx context.Context, dto LoginRequest) (*AuthResponse, error) {
// 	existingUser, err := s.userRepository.GetByEmail(ctx, dto.Email)
// 	if err != nil {
// 		s.logger.Error("internal error when fetching user", zap.Error(err))
// 		return nil, errors.New("internal error when fetching user")
// 	}

// 	if existingUser == nil {
// 		return nil, errors.New("invalid credentials")
// 	}

// 	s.logger.Info("info", zap.String("email", existingUser.Email), zap.String("pass", string(existingUser.PasswordHash)))

// 	if err := bcrypt.CompareHashAndPassword(existingUser.PasswordHash, []byte(dto.Password)); err != nil {
// 		return nil, errors.New("invalid credentials")
// 	}

// 	token, err := s.tokenManager.GenerateToken(existingUser.ID)
// 	if err != nil {
// 		s.logger.Error("internal error when generating token", zap.Error(err))
// 		return nil, errors.New("internal error")
// 	}

// 	return &AuthResponse{
// 		User:       *existingUser,
// 		AcessToken: token,
// 	}, nil
// }
