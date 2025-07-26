package service

import (
	"context"

	"github.com/vetrovegor/kushfinds-backend/internal/industry"
	"go.uber.org/zap"
)

type Repository interface {
	GetAll(ctx context.Context) ([]industry.Industry, error)
}

type service struct {
	repository   Repository
	logger       *zap.Logger
}

func New(
	repository Repository,
	logger *zap.Logger,
) *service {
	return &service{
		repository:   repository,
		logger:       logger,
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
