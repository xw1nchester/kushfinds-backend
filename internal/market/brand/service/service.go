package service

import (
	"context"

	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/location/country"
	"github.com/vetrovegor/kushfinds-backend/internal/market/brand"
	"go.uber.org/zap"
)

var (
	ErrBrandNameAlreadyExists = apperror.NewAppError("the brand with this name already exists")
)

type Repository interface {
	CheckBrandNameIsAvailable(ctx context.Context, name string) (bool, error)
	GetBrandsByUserID(ctx context.Context, id int) ([]brand.Brand, error)
	GetBrandByID(ctx context.Context, id int) (*brand.Brand, error)
	CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error)
}

type UserService interface {
	CheckBusinessProfileExists(ctx context.Context, userID int) error
}

type CountryService interface {
	GetByID(ctx context.Context, id int) (*country.Country, error)
}

type StateService interface {
	CheckStatesExist(ctx context.Context, stateIDs []int) error
}

type MarketSectionService interface {
	CheckStatesExist(ctx context.Context, marketSectionIDs []int) error
}

type service struct {
	repository           Repository
	userService          UserService
	countryService       CountryService
	stateService         StateService
	marketSectionService MarketSectionService
	logger               *zap.Logger
}

func New(
	repository Repository,
	userService UserService,
	countryService CountryService,
	stateService StateService,
	marketSectionService MarketSectionService,
	logger *zap.Logger,
) *service {
	return &service{
		repository:           repository,
		userService:          userService,
		countryService:       countryService,
		stateService:         stateService,
		marketSectionService: marketSectionService,
		logger:               logger,
	}
}

func (s *service) CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error) {
	if err := s.userService.CheckBusinessProfileExists(ctx, data.UserID); err != nil {
		return nil, err
	}

	if _, err := s.countryService.GetByID(ctx, data.Country.ID); err != nil {
		return nil, err
	}

	stateIDs := make([]int, len(data.States))
	for i, s := range data.States {
		stateIDs[i] = s.ID
	}

	if err := s.stateService.CheckStatesExist(ctx, stateIDs); err != nil {
		return nil, err
	}

	marketSectionIDs := make([]int, len(data.MarketSubSections)+1)
	marketSectionIDs[0] = data.MarketSection.ID
	for i, ms := range data.MarketSubSections {
		marketSectionIDs[i+1] = ms.ID
	}

	if err := s.marketSectionService.CheckStatesExist(ctx, marketSectionIDs) ; err != nil {
		return nil, err
	}

	nameIsAvailable, err := s.repository.CheckBrandNameIsAvailable(ctx, data.Name)
	if !nameIsAvailable {
		if err != nil {
			s.logger.Error("unexpected error when checking brand name availability", zap.Error(err))
			return nil, err
		} else {
			return nil, ErrBrandNameAlreadyExists
		}
	}

	createdBrand, err := s.repository.CreateBrand(ctx, data)
	if err != nil {
		s.logger.Error("unexpected error when creating brand", zap.Error(err))
		return nil, err
	}

	return createdBrand, nil
}
