package storedb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xw1nchester/kushfinds-backend/internal/logging"
	"github.com/xw1nchester/kushfinds-backend/internal/market/store"
	"go.uber.org/zap"
)

type repository struct {
	client *pgxpool.Pool
	logger *zap.Logger
}

func New(client *pgxpool.Pool, logger *zap.Logger) *repository {
	return &repository{
		client: client,
		logger: logger,
	}
}

func (r *repository) GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error) {
	query := `SELECT id, name FROM store_types`

	logging.LogSQLQuery(r.logger, query)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	storeTypes := make([]store.StoreType, 0)
	for rows.Next() {
		var storeType store.StoreType

		err := rows.Scan(
			&storeType.ID,
			&storeType.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		storeTypes = append(storeTypes, storeType)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return storeTypes, nil
}

func (r *repository) GetStoreTypeByID(ctx context.Context, id int) (*store.StoreType, error) {
	query := `
        SELECT id, name FROM store_types
		WHERE id=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var storeType store.StoreType
	err := r.client.QueryRow(ctx, query, id).Scan(&storeType.ID, &storeType.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStoreTypeNotFound
		}
		return nil, err
	}

	return &storeType, nil
}