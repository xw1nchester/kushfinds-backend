package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xw1nchester/kushfinds-backend/internal/logging"
	"github.com/xw1nchester/kushfinds-backend/internal/market/industry"
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

func (r *repository) GetByID(ctx context.Context, id int) (*industry.Industry, error) {
	query := `
        SELECT id, name FROM business_industries
		WHERE id=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var industry industry.Industry
	err := r.client.QueryRow(ctx, query, id).Scan(&industry.ID, &industry.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrIndustryNotFound
		}
		return nil, err
	}

	return &industry, nil
}
