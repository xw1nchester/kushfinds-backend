package storeservice

import (
	"context"
	"errors"

	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	"github.com/xw1nchester/kushfinds-backend/internal/market/store"
	storedb "github.com/xw1nchester/kushfinds-backend/internal/market/store/db"
	"go.uber.org/zap"
)

type Repository interface {
	GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error)
	GetStoreTypeByID(ctx context.Context, id int) (*store.StoreType, error)

	CreateStore(ctx context.Context, data store.Store) (*store.Store, error)
}

type UserService interface {
	CheckBusinessProfileExists(ctx context.Context, userID int) error
}

type BrandService interface {
	CheckBrandExists(ctx context.Context, brandID, userID int) error
}

type CountryService interface {
	GetByID(ctx context.Context, id int) (*country.Country, error)
}

type StateService interface {
	GetByID(ctx context.Context, id int) (*state.State, error)
}

type RegionService interface {
	GetByID(ctx context.Context, id int) (*region.Region, error)
}

type service struct {
	repository     Repository
	userService    UserService
	brandService   BrandService
	countryService CountryService
	stateService   StateService
	regionService  RegionService
	logger         *zap.Logger
}

func New(
	repository Repository,
	userService UserService,
	brandService BrandService,
	countryService CountryService,
	stateService StateService,
	regionService RegionService,
	logger *zap.Logger,
) *service {
	return &service{
		repository:     repository,
		userService:    userService,
		brandService:   brandService,
		countryService: countryService,
		stateService:   stateService,
		regionService:  regionService,
		logger:         logger,
	}
}

func (s *service) GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error) {
	storeTypes, err := s.repository.GetAllStoreTypes(ctx)
	if err != nil {
		s.logger.Error("unexpected error when fetching all store types", zap.Error(err))

		return nil, err
	}

	return storeTypes, nil
}

func (s *service) validateStoreData(ctx context.Context, data store.Store) error {
	if err := s.userService.CheckBusinessProfileExists(ctx, data.UserID); err != nil {
		return err
	}

	if err := s.brandService.CheckBrandExists(ctx, data.Brand.ID, data.UserID); err != nil {
		return err
	}

	if _, err := s.countryService.GetByID(ctx, data.Country.ID); err != nil {
		return err
	}

	if _, err := s.stateService.GetByID(ctx, data.State.ID); err != nil {
		return err
	}

	if _, err := s.regionService.GetByID(ctx, data.Region.ID); err != nil {
		return err
	}

	if _, err := s.repository.GetStoreTypeByID(ctx, data.StoreType.ID); err != nil {
		if errors.Is(err, storedb.ErrStoreTypeNotFound) {
			return apperror.ErrNotFound
		}
		s.logger.Error("unexpected error when fetching store type by id", zap.Error(err))
		return err
	}

	return nil
}

func (s *service) CreateStore(ctx context.Context, data store.Store) (*store.Store, error) {
	if err := s.validateStoreData(ctx, data); err != nil {
		return nil, err
	}

	createdStore, err := s.repository.CreateStore(ctx, data)
	if err != nil {
		s.logger.Error("unexpected error when creating store", zap.Error(err))
		return nil, err
	}

	return createdStore, nil
}
