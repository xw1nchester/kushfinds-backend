package service

import (
	"context"
	"errors"

	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state/db"
	"go.uber.org/zap"
)

type Repository interface {
	GetAll(ctx context.Context) ([]*db.State, error)
	GetByID(ctx context.Context, id int) (*db.State, error)
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
