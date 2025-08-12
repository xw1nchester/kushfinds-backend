package service

import (
	"context"
	"errors"

	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/country/db"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	"go.uber.org/zap"
)

type Repository interface {
	GetAll(ctx context.Context) ([]country.Country, error)
	GetByID(ctx context.Context, id int) (*country.Country, error)
}

type StateService interface {
	GetAllByCountryID(ctx context.Context, id int) ([]state.State, error)
}

type service struct {
	repository   Repository
	stateService StateService
	logger       *zap.Logger
}

func New(
	repository Repository,
	stateService StateService,
	logger *zap.Logger,
) *service {
	return &service{
		repository:   repository,
		stateService: stateService,
		logger:       logger,
	}
}

func (s *service) GetAll(ctx context.Context) ([]country.Country, error) {
	countries, err := s.repository.GetAll(ctx)
	if err != nil {
		s.logger.Error("unexpected error when fetching all countries", zap.Error(err))

		return nil, err
	}

	return countries, nil
}

func (s *service) GetByID(ctx context.Context, id int) (*country.Country, error) {
	existingCountry, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrCountryNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching country by id", zap.Error(err))

		return nil, err
	}

	return &country.Country{
		ID:   existingCountry.ID,
		Name: existingCountry.Name,
	}, nil
}

func (s *service) GetCountryStates(ctx context.Context, id int) ([]state.State, error) {
	return s.stateService.GetAllByCountryID(ctx, id)
}
