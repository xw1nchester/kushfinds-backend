package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xw1nchester/kushfinds-backend/internal/industry"
	"github.com/xw1nchester/kushfinds-backend/internal/logging"
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

func (r *repository) GetAll(ctx context.Context) ([]industry.Industry, error) {
	query := `SELECT id, name FROM business_industries`

	logging.LogSQLQuery(r.logger, query)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	industries := make([]industry.Industry, 0)
	for rows.Next() {
		var industry industry.Industry

		err := rows.Scan(
			&industry.ID,
			&industry.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		industries = append(industries, industry)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return industries, nil
}
