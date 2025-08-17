package storeservice

import (
	"context"

	"github.com/xw1nchester/kushfinds-backend/internal/market/store"
	"go.uber.org/zap"
)

type Repository interface {
	GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error)
	GetStoreTypeByID(ctx context.Context, id int) (*store.StoreType, error)
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

func (s *service) GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error) {
	storeTypes, err := s.repository.GetAllStoreTypes(ctx)
	if err != nil {
		s.logger.Error("unexpected error when fetching all store types", zap.Error(err))

		return nil, err
	}

	return storeTypes, nil
}