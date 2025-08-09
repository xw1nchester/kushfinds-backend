package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vetrovegor/kushfinds-backend/internal/logging"
	marketsection "github.com/vetrovegor/kushfinds-backend/internal/market/section"
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

func (r *repository) GetAll(ctx context.Context) ([]marketsection.MarketSection, error) {
	query := `SELECT id, name FROM market_sections`

	logging.LogSQLQuery(r.logger, query)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	marketSections := make([]marketsection.MarketSection, 0)
	for rows.Next() {
		var marketSection marketsection.MarketSection

		err := rows.Scan(
			&marketSection.ID,
			&marketSection.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		marketSections = append(marketSections, marketSection)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return marketSections, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*marketsection.MarketSection, error) {
	query := `
        SELECT id, name FROM market_sections
		WHERE id=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var marketSection marketsection.MarketSection
	err := r.client.QueryRow(ctx, query, id).Scan(&marketSection.ID, &marketSection.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrMarketSectionNotFound
		}
		return nil, err
	}

	return &marketSection, nil
}
