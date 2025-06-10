package auth

import (
	"context"
	"errors"

	"github.com/vetrovegor/kushfinds-backend/internal/config"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repository user.Repository
	jwtConfig  config.JWT
	logger     *zap.Logger
}

func NewService(repository user.Repository, jwtConfig config.JWT, logger *zap.Logger) Service {
	return Service{
		repository: repository,
		jwtConfig:  jwtConfig,
		logger:     logger,
	}
}

func (s *Service) Register(ctx context.Context, dto RegisterRequest) (*AuthResponse, error) {
	existingUser, err := s.repository.GetByEmail(ctx, dto.Email)
	if err != nil {
		s.logger.Error("unexpected error when fetching user", zap.Error(err))
		return nil, errors.New("unexpected error when fetching user")
	}

	if existingUser != nil {
		return nil, errors.New("the user with this email already exists")
	}

	passHash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("unexpected error when hashing password", zap.Error(err))
		return nil, errors.New("unexpected error when hashing password")
	}

	// TODO: в дальнейшем убрать хардкод роли
	createdUser, err := s.repository.Create(ctx, user.User{Email: dto.Email, PasswordHash: passHash, Role: "buyerr"})
	if err != nil {
		s.logger.Error("unexpected error when creating user", zap.Error(err))
		return nil, errors.New("unexpected error when creating user")
	}

	token, err := GenerateToken(s.jwtConfig, existingUser.ID)
	if err != nil {
		s.logger.Error("unexpected error when generating token", zap.Error(err))
		return nil, errors.New("unexpected error")
	}

	return &AuthResponse{
		User:       *createdUser,
		AcessToken: token,
	}, nil
}

func (s *Service) Login(ctx context.Context, dto LoginRequest) (*AuthResponse, error) {
	existingUser, err := s.repository.GetByEmail(ctx, dto.Email)
	if err != nil {
		s.logger.Error("unexpected error when fetching user", zap.Error(err))
		return nil, errors.New("unexpected error when fetching user")
	}

	if existingUser == nil {
		return nil, errors.New("invalid credentials")
	}

	s.logger.Info("info", zap.String("email", existingUser.Email), zap.String("pass", string(existingUser.PasswordHash)))

	if err := bcrypt.CompareHashAndPassword(existingUser.PasswordHash, []byte(dto.Password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, err := GenerateToken(s.jwtConfig, existingUser.ID)
	if err != nil {
		s.logger.Error("unexpected error when generating token", zap.Error(err))
		return nil, errors.New("unexpected error")
	}

	return &AuthResponse{
		User:       *existingUser,
		AcessToken: token,
	}, nil
}
