package storeservice

import (
	"context"
	"errors"

	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/market/store"
	storedb "github.com/xw1nchester/kushfinds-backend/internal/market/store/db"
	"go.uber.org/zap"
)

type Repository interface {
	GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error)
	GetStoreTypeByID(ctx context.Context, id int) (*store.StoreType, error)

	CreateStore(ctx context.Context, data store.Store) (*store.Store, error)
	GetUserStores(ctx context.Context, userID int) ([]store.StoreSummary, error)
	GetStoreByID(ctx context.Context, id int) (*store.Store, error)
}

type UserService interface {
	CheckBusinessProfileExists(ctx context.Context, userID int, requireVerified bool) error
}

type BrandService interface {
	CheckBrandExists(ctx context.Context, brandID, userID int) error
}

type RegionService interface {
	CheckLocationExists(ctx context.Context, regionID, stateID, countryID int) error
}

type SocialService interface {
	CheckSocialsExist(ctx context.Context, IDs []int) error
}

type service struct {
	repository    Repository
	userService   UserService
	brandService  BrandService
	regionService RegionService
	socialService SocialService
	logger        *zap.Logger
}

func New(
	repository Repository,
	userService UserService,
	brandService BrandService,
	regionService RegionService,
	socialService SocialService,
	logger *zap.Logger,
) *service {
	return &service{
		repository:    repository,
		userService:   userService,
		brandService:  brandService,
		regionService: regionService,
		socialService: socialService,
		logger:        logger,
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
	if err := s.userService.CheckBusinessProfileExists(
		ctx,
		data.UserID,
		data.IsPublished,
	); err != nil {
		return err
	}

	if err := s.brandService.CheckBrandExists(ctx, data.Brand.ID, data.UserID); err != nil {
		return err
	}

	if err := s.regionService.CheckLocationExists(
		ctx,
		data.Region.ID,
		data.State.ID,
		data.Country.ID,
	); err != nil {
		return err
	}

	socialIDs := make([]int, len(data.Socials))
	for i, s := range data.Socials {
		socialIDs[i] = s.ID
	}

	if err := s.socialService.CheckSocialsExist(ctx, socialIDs); err != nil {
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

func (s *service) GetUserStores(ctx context.Context, userID int) ([]store.StoreSummary, error) {
	brands, err := s.repository.GetUserStores(ctx, userID)
	if err != nil {
		s.logger.Error("unexpected error when fetching user stores", zap.Error(err))

		return nil, err
	}

	return brands, nil
}

func (s *service) GetUserStore(ctx context.Context, storeID, userID int) (*store.Store, error) {
	store, err := s.repository.GetStoreByID(ctx, storeID)
	if err != nil {
		if errors.Is(err, storedb.ErrStoreNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching store by id", zap.Error(err))

		return nil, err
	}

	if store.UserID != userID {
		return nil, apperror.ErrNotFound
	}

	return store, nil
}
