package service

import (
	"context"
	"errors"

	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/market/brand"
	"github.com/xw1nchester/kushfinds-backend/internal/market/brand/db"
	"go.uber.org/zap"
)

var (
	ErrBrandNameAlreadyExists = apperror.NewAppError("the brand with this name already exists")
)

type Repository interface {
	CheckBrandNameIsAvailable(ctx context.Context, name string, excludeID ...int) (bool, error)
	GetUserBrands(ctx context.Context, userID int) ([]brand.BrandSummary, error)
	GetUserBrand(ctx context.Context, brandID, userID int) (*brand.Brand, error)
	CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error)
	CheckBrandExists(ctx context.Context, brandID, userID int) error
	UpdateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error)
	DeleteBrand(ctx context.Context, brandID, userID int) error
}

type UserService interface {
	CheckBusinessProfileExists(ctx context.Context, userID int, requireVerified bool) error
}

type CountryService interface {
	GetByID(ctx context.Context, id int) (*country.Country, error)
}

type StateService interface {
	CheckStatesExist(ctx context.Context, stateIDs []int) error
}

type MarketSectionService interface {
	CheckMarketSectionsExist(ctx context.Context, IDs []int) error
}

type SocialService interface {
	CheckSocialsExist(ctx context.Context, IDs []int) error
}

type service struct {
	repository           Repository
	userService          UserService
	countryService       CountryService
	stateService         StateService
	marketSectionService MarketSectionService
	socialService        SocialService
	logger               *zap.Logger
}

func New(
	repository Repository,
	userService UserService,
	countryService CountryService,
	stateService StateService,
	marketSectionService MarketSectionService,
	socialService SocialService,
	logger *zap.Logger,
) *service {
	return &service{
		repository:           repository,
		userService:          userService,
		countryService:       countryService,
		stateService:         stateService,
		marketSectionService: marketSectionService,
		socialService:        socialService,
		logger:               logger,
	}
}

func (s *service) CheckBrandExists(ctx context.Context, brandID, userID int) error {
	err := s.repository.CheckBrandExists(ctx, brandID, userID)
	if err != nil {
		if errors.Is(err, db.ErrBrandNotFound) {
			return apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when check brand exists by id", zap.Error(err))
	}

	return err
}

func (s *service) validateBrandData(ctx context.Context, data brand.Brand, isUpdate bool) error {
	if isUpdate {
		if err := s.CheckBrandExists(ctx, data.ID, data.UserID); err != nil {
			return err
		}
	}

	if err := s.userService.CheckBusinessProfileExists(
		ctx,
		data.UserID,
		data.IsPublished,
	); err != nil {
		return err
	}

	if _, err := s.countryService.GetByID(ctx, data.Country.ID); err != nil {
		return err
	}

	stateIDs := make([]int, len(data.States))
	for i, s := range data.States {
		stateIDs[i] = s.ID
	}

	if err := s.stateService.CheckStatesExist(ctx, stateIDs); err != nil {
		return err
	}

	marketSectionIDs := make([]int, 0)
	marketSectionIDs = append(marketSectionIDs, data.MarketSection.ID)
	for _, ms := range data.MarketSubSections {
		if ms.ID != data.MarketSection.ID {
			marketSectionIDs = append(marketSectionIDs, ms.ID)
		}
	}

	if err := s.marketSectionService.CheckMarketSectionsExist(ctx, marketSectionIDs); err != nil {
		return err
	}

	socialIDs := make([]int, len(data.Socials))
	for i, s := range data.Socials {
		socialIDs[i] = s.ID
	}

	if err := s.socialService.CheckSocialsExist(ctx, socialIDs); err != nil {
		return err
	}

	args := []int{}

	if isUpdate {
		args = append(args, data.ID)
	}

	nameIsAvailable, err := s.repository.CheckBrandNameIsAvailable(ctx, data.Name, args...)
	if !nameIsAvailable {
		if err != nil {
			s.logger.Error("unexpected error when checking brand name availability", zap.Error(err))
			return err
		} else {
			return ErrBrandNameAlreadyExists
		}
	}

	return nil
}

func (s *service) CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error) {
	err := s.validateBrandData(ctx, data, false)
	if err != nil {
		return nil, err
	}

	createdBrand, err := s.repository.CreateBrand(ctx, data)
	if err != nil {
		s.logger.Error("unexpected error when creating brand", zap.Error(err))
		return nil, err
	}

	return createdBrand, nil
}

func (s *service) GetUserBrands(ctx context.Context, userID int) ([]brand.BrandSummary, error) {
	brands, err := s.repository.GetUserBrands(ctx, userID)
	if err != nil {
		s.logger.Error("unexpected error when fetching user brands", zap.Error(err))

		return nil, err
	}

	return brands, nil
}

func (s *service) GetUserBrand(ctx context.Context, brandID, userID int) (*brand.Brand, error) {
	brand, err := s.repository.GetUserBrand(ctx, brandID, userID)
	if err != nil {
		if errors.Is(err, db.ErrBrandNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching brand by id", zap.Error(err))

		return nil, err
	}

	return brand, nil
}

func (s *service) UpdateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error) {
	err := s.validateBrandData(ctx, data, true)
	if err != nil {
		return nil, err
	}

	updatedBrand, err := s.repository.UpdateBrand(ctx, data)
	if err != nil {
		s.logger.Error("unexpected error when updating brand", zap.Error(err))
		return nil, err
	}

	return updatedBrand, nil
}

func (s *service) DeleteBrand(ctx context.Context, brandID, userID int) error {
	if err := s.CheckBrandExists(ctx, brandID, userID); err != nil {
		return err
	}

	err := s.repository.DeleteBrand(ctx, brandID, userID)
	if err != nil {
		s.logger.Error("unexpected error when deleting brand", zap.Error(err))
	}

	return err
}
