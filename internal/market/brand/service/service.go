package service

import (
	"context"

	"github.com/vetrovegor/kushfinds-backend/internal/location/country"
	"github.com/vetrovegor/kushfinds-backend/internal/market/brand"
	"go.uber.org/zap"
)

type Repository interface {
	GetBrandsByUserID(ctx context.Context, id int) ([]brand.Brand, error)
	GetBrandByID(ctx context.Context, id int) (*brand.Brand, error)
	CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error)
}

type CountryService interface {
	GetByID(ctx context.Context, id int) (*country.Country, error)
}

type StateService interface {
	CheckStatesExist(ctx context.Context, stateIDs []int) error
}

type service struct {
	repository     Repository
	countryService CountryService
	stateService   StateService
	logger         *zap.Logger
}

func New(
	repository Repository,
	countryService CountryService,
	stateService StateService,
	logger *zap.Logger,
) *service {
	return &service{
		repository:     repository,
		countryService: countryService,
		stateService:   stateService,
		logger:         logger,
	}
}

func (s *service) CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error) {
	// TODO: check business profile exists
	// check market sections exists
	// check brand name doesn't exists

	if _, err := s.countryService.GetByID(ctx, data.Country.ID); err != nil {
		return nil, err
	}

	stateIDs := make([]int, len(data.States))
	for _, state := range data.States {
		stateIDs = append(stateIDs, state.ID)
	}

	if err := s.stateService.CheckStatesExist(ctx, stateIDs); err != nil {
		return nil, err
	}

	createdBrand, err := s.repository.CreateBrand(ctx, data)
	if err != nil {
		s.logger.Error("unexpected error when creating brand", zap.Error(err))
		return nil, err
	}

	return createdBrand, nil
}
