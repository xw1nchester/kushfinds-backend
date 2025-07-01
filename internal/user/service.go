package user

import (
	"context"
	"errors"

	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/user/db"
	"go.uber.org/zap"
)

//go:generate mockgen -source=service.go -destination=mocks/mock.go -package=mockuserservice
type Service interface {
	GetByID(ctx context.Context, id int) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Create(ctx context.Context, email string) (int, error)
	Verify(ctx context.Context, id int) (*User, error)
	CheckUsernameIsAvailable(ctx context.Context, username string) (bool, error)
	SetProfileInfo(ctx context.Context, data *User) (*User, error)
	SetPassword(ctx context.Context, id int, passwordHash []byte) error
}

type service struct {
	repository db.Repository
	logger     *zap.Logger
}

func NewService(
	repository db.Repository,
	logger *zap.Logger,
) Service {
	return &service{
		repository: repository,
		logger:     logger,
	}
}

func (s *service) GetByID(ctx context.Context, id int) (*User, error) {
	user, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching user by id", zap.Error(err))

		return nil, err
	}

	return &User{
		ID:            user.ID,
		Email:         user.Email,
		Username:      user.Username,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Avatar:        user.Avatar,
		IsVerified:    user.IsVerified,
		PasswordHash:  user.PasswordHash,
		IsPasswordSet: user.PasswordHash != nil,
	}, nil
}

func (s *service) GetByEmail(ctx context.Context, email string) (*User, error) {
	user, err := s.repository.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching user by email", zap.Error(err))

		return nil, err
	}

	return &User{
		ID:            user.ID,
		Email:         user.Email,
		Username:      user.Username,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Avatar:        user.Avatar,
		IsVerified:    user.IsVerified,
		PasswordHash:  user.PasswordHash,
		IsPasswordSet: user.PasswordHash != nil,
	}, nil
}

func (s *service) Create(ctx context.Context, email string) (int, error) {
	userID, err := s.repository.Create(ctx, email)
	if err != nil {
		s.logger.Error("unexpected error when creating user", zap.Error(err))
		return 0, err
	}

	return userID, nil
}

func (s *service) Verify(ctx context.Context, id int) (*User, error) {
	user, err := s.repository.Verify(ctx, id)
	if err != nil {
		s.logger.Error("unexpected error when verifying user", zap.Error(err))
		return nil, err
	}

	return &User{
		ID:            user.ID,
		Email:         user.Email,
		Username:      user.Username,
		FirstName:     user.FirstName,
		LastName:      user.LastName,
		Avatar:        user.Avatar,
		IsVerified:    user.IsVerified,
		PasswordHash:  user.PasswordHash,
		IsPasswordSet: user.PasswordHash != nil,
	}, nil
}

func (s *service) CheckUsernameIsAvailable(ctx context.Context, username string) (bool, error) {
	isAvailable, err := s.repository.CheckUsernameIsAvailable(ctx, username)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return isAvailable, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching user by username", zap.Error(err))
	}

	return isAvailable, err
}

func (s *service) SetProfileInfo(ctx context.Context, user *User) (*User, error) {
	updatedUser, err := s.repository.SetProfileInfo(
		ctx,
		db.User{
			ID:        user.ID,
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
		},
	)
	if err != nil {
		s.logger.Error("unexpected error when updating user profile", zap.Error(err))
		return nil, err
	}

	return &User{
		ID:            updatedUser.ID,
		Email:         updatedUser.Email,
		Username:      updatedUser.Username,
		FirstName:     updatedUser.FirstName,
		LastName:      updatedUser.LastName,
		Avatar:        updatedUser.Avatar,
		IsVerified:    updatedUser.IsVerified,
		PasswordHash:  updatedUser.PasswordHash,
		IsPasswordSet: updatedUser.PasswordHash != nil,
	}, nil
}

func (s *service) SetPassword(ctx context.Context, id int, passwordHash []byte) error {
	if err := s.repository.SetPassword(ctx, id, passwordHash); err != nil {
		s.logger.Error("unexpected error when set user password", zap.Error(err))
		return err
	}

	return nil
}
