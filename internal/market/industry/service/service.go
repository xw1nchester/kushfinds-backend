package service

import (
	"context"
	"errors"

	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/market/industry"
	"github.com/xw1nchester/kushfinds-backend/internal/market/industry/db"
	"go.uber.org/zap"
)

type Repository interface {
	GetAll(ctx context.Context) ([]industry.Industry, error)
	GetByID(ctx context.Context, id int) (*industry.Industry, error)
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

func (s *service) GetAll(ctx context.Context) ([]industry.Industry, error) {
	industries, err := s.repository.GetAll(ctx)
	if err != nil {
		s.logger.Error("unexpected error when fetching all industries", zap.Error(err))

		return nil, err
	}

	return industries, nil
}

func (s *service) GetByID(ctx context.Context, id int) (*industry.Industry, error) {
	existingIndustry, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrIndustryNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching business industry by id", zap.Error(err))

		return nil, err
	}

	return existingIndustry, nil
}
