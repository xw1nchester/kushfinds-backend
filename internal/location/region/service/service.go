package service

import (
	"context"
	"errors"

	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region/db"
	"go.uber.org/zap"
)

type Repository interface {
	GetAll(ctx context.Context) ([]region.Region, error)
	GetByID(ctx context.Context, id int) (*region.Region, error)
	GetAllByStateID(ctx context.Context, countryID int) ([]region.Region, error)
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

func (s *service) GetByID(ctx context.Context, id int) (*region.Region, error) {
	existingRegion, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrRegionNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching region by id", zap.Error(err))

		return nil, err
	}

	return &region.Region{
		ID:   existingRegion.ID,
		Name: existingRegion.Name,
	}, nil
}

func (s *service) GetAllByStateID(ctx context.Context, id int) ([]region.Region, error) {
	regions, err := s.repository.GetAllByStateID(ctx, id)
	if err != nil {
		s.logger.Error("unexpected error when fetching regions by state id", zap.Error(err))

		return nil, err
	}

	return regions, nil
}
