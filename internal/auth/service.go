package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	authDB "github.com/vetrovegor/kushfinds-backend/internal/auth/db"
	"github.com/vetrovegor/kushfinds-backend/internal/code"
	userDB "github.com/vetrovegor/kushfinds-backend/internal/user/db"
	"go.uber.org/zap"
)

var (
	ErrInternal          = errors.New("unexpected error")
	ErrUserAlreadyExists = errors.New("the user with this email already exists")
)

type Service interface {
	RegisterEmail(ctx context.Context, dto RegisterEmailRequest, userAgent string) (*Tokens, error)
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
func (s service) RegisterEmail(ctx context.Context, dto RegisterEmailRequest, userAgent string) (*Tokens, error) {
	_, err := s.userRepository.GetByEmail(ctx, dto.Email)
	if err != nil && !errors.Is(err, userDB.ErrUserNotFound) {
		s.logger.Error("error when fetching user", zap.Error(err))
		return nil, ErrInternal
	}

	if err == nil {
		return nil, ErrUserAlreadyExists
	}

	userID, err := s.userRepository.Create(ctx, dto.Email)
	if err != nil {
		s.logger.Error("error when creating user", zap.Error(err))
		return nil, ErrInternal
	}

	code, err := s.codeService.GenerateVerify(ctx, userID)
	if err != nil {
		return nil, ErrInternal
	}

	go func() {
		if err := s.mailManager.SendMail(
			"Confirmation of registration",
			fmt.Sprintf("Your registration confirmation code: %s", code),
			[]string{dto.Email},
		); err != nil {
			s.logger.Error("error when sending email", zap.Error(err))
		}
	}()

	accessToken, err := s.tokenManager.GenerateToken(userID)
	if err != nil {
		s.logger.Error("error when generating jwt token", zap.Error(err))

		return nil, ErrInternal
	}

	refreshToken, err := s.generateRefreshToken(ctx, userAgent, userID)
	if err != nil {
		s.logger.Error("error when generating jwt token", zap.Error(err))
		return nil, ErrInternal
	}

	return &Tokens{JwtToken{AccessToken: accessToken}, refreshToken}, nil
}

// func (s Service) Register(ctx context.Context, dto RegisterRequest) (*AuthResponse, error) {
// 	existingUser, err := s.userRepository.GetByEmail(ctx, dto.Email)
// 	if err != nil {
// 		s.logger.Error("unexpected error when fetching user", zap.Error(err))
// 		return nil, errors.New("unexpected error when fetching user")
// 	}

// 	if existingUser != nil {
// 		return nil, errors.New("the user with this email already exists")
// 	}

// 	passHash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
// 	if err != nil {
// 		s.logger.Error("unexpected error when hashing password", zap.Error(err))
// 		return nil, errors.New("unexpected error when hashing password")
// 	}

// 	// TODO: в дальнейшем убрать хардкод роли
// 	createdUser, err := s.userRepository.Create(ctx, user.User{Email: dto.Email, PasswordHash: passHash, Role: "buyer"})
// 	if err != nil {
// 		s.logger.Error("unexpected error when creating user", zap.Error(err))
// 		return nil, errors.New("unexpected error when creating user")
// 	}

// 	token, err := s.tokenManager.GenerateToken(createdUser.ID)
// 	if err != nil {
// 		s.logger.Error("unexpected error when generating token", zap.Error(err))
// 		return nil, errors.New("unexpected error")
// 	}

// 	return &AuthResponse{
// 		User:       *createdUser,
// 		AcessToken: token,
// 	}, nil
// }

// func (s Service) Login(ctx context.Context, dto LoginRequest) (*AuthResponse, error) {
// 	existingUser, err := s.userRepository.GetByEmail(ctx, dto.Email)
// 	if err != nil {
// 		s.logger.Error("unexpected error when fetching user", zap.Error(err))
// 		return nil, errors.New("unexpected error when fetching user")
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
// 		s.logger.Error("unexpected error when generating token", zap.Error(err))
// 		return nil, errors.New("unexpected error")
// 	}

// 	return &AuthResponse{
// 		User:       *existingUser,
// 		AcessToken: token,
// 	}, nil
// }
