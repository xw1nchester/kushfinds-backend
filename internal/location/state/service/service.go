package service

import (
	"context"
	"errors"

	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/location/region"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state/db"
	"go.uber.org/zap"
)

type Repository interface {
	GetAll(ctx context.Context) ([]state.State, error)
	GetByID(ctx context.Context, id int) (*state.State, error)
	GetAllByCountryID(ctx context.Context, countryID int) ([]state.State, error)
}

type RegionService interface {
	GetAllByStateID(ctx context.Context, countryID int) ([]region.Region, error)
}

type service struct {
	repository    Repository
	regionService RegionService
	logger        *zap.Logger
}

func New(
	repository Repository,
	regionService RegionService,
	logger *zap.Logger,
) *service {
	return &service{
		repository:    repository,
		regionService: regionService,
		logger:        logger,
	}
}

func (s *service) GetByID(ctx context.Context, id int) (*state.State, error) {
	existingState, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrStateNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching state by id", zap.Error(err))

		return nil, err
	}

	return &state.State{
		ID:   existingState.ID,
		Name: existingState.Name,
	}, nil
}

// TODO: в дальнейшем реализовать GetAll, который может принимать фильтры
func (s *service) GetAllByCountryID(ctx context.Context, id int) ([]state.State, error) {
	states, err := s.repository.GetAllByCountryID(ctx, id)
	if err != nil {
		s.logger.Error("unexpected error when fetching states by country id", zap.Error(err))

		return nil, err
	}

	return states, nil
}

func (s *service) GetStateRegions(ctx context.Context, id int) ([]region.Region, error) {
	return s.regionService.GetAllByStateID(ctx, id)
}
