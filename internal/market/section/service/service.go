package service

import (
	"context"
	"errors"

	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	marketsection "github.com/vetrovegor/kushfinds-backend/internal/market/section"
	"github.com/vetrovegor/kushfinds-backend/internal/market/section/db"
	"go.uber.org/zap"
)

type Repository interface {
	GetAll(ctx context.Context) ([]marketsection.MarketSection, error)
	GetByID(ctx context.Context, id int) (*marketsection.MarketSection, error)
	CheckMarketSectionsExist(ctx context.Context, marketSectionIDs []int) error
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

func (s *service) GetAll(ctx context.Context) ([]marketsection.MarketSection, error) {
	marketSections, err := s.repository.GetAll(ctx)
	if err != nil {
		s.logger.Error("unexpected error when fetching all market sections", zap.Error(err))

		return nil, err
	}

	return marketSections, nil
}

func (s *service) GetByID(ctx context.Context, id int) (*marketsection.MarketSection, error) {
	existingMarketSection, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrMarketSectionNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching market section by id", zap.Error(err))

		return nil, err
	}

	return &marketsection.MarketSection{
		ID:   existingMarketSection.ID,
		Name: existingMarketSection.Name,
	}, nil
}

func (s *service) CheckStatesExist(ctx context.Context, marketSectionIDs []int) error {
	err := s.repository.CheckMarketSectionsExist(ctx, marketSectionIDs)
	if err != nil {
		if errors.Is(err, db.ErrMarketSectionNotFound) {
			return apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when check market sections exists", zap.Error(err))
	}

	return err
}
