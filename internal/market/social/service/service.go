package socialservice

import (
	"context"
	"errors"

	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/market/social"
	socialdb "github.com/xw1nchester/kushfinds-backend/internal/market/social/db"
	"go.uber.org/zap"
)

type Repository interface {
	GetAll(ctx context.Context) ([]social.Social, error)
	GetByID(ctx context.Context, id int) (*social.Social, error)
	CheckSocialsExist(ctx context.Context, IDs []int) error
}

type service struct {
	repository Repository
	logger     *zap.Logger
}

func New(
	repository Repository,
	logger *zap.Logger,
) *service {
	return &service{
		repository: repository,
		logger:     logger,
	}
}

func (s *service) GetAll(ctx context.Context) ([]social.Social, error) {
	socials, err := s.repository.GetAll(ctx)
	if err != nil {
		s.logger.Error("unexpected error when fetching all socials", zap.Error(err))

		return nil, err
	}

	return socials, nil
}

func (s *service) GetByID(ctx context.Context, id int) (*social.Social, error) {
	existingSocial, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, socialdb.ErrSocialNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching social by id", zap.Error(err))

		return nil, err
	}

	return existingSocial, nil
}

func (s *service) CheckSocialsExist(ctx context.Context, IDs []int) error {
	err := s.repository.CheckSocialsExist(ctx, IDs)
	if err != nil {
		if errors.Is(err, socialdb.ErrSocialNotFound) {
			return apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when check socials exists", zap.Error(err))
	}

	return err
}
