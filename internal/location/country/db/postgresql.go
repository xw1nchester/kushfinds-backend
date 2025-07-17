package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vetrovegor/kushfinds-backend/internal/location/country"
	"github.com/vetrovegor/kushfinds-backend/internal/logging"
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

func (r *repository) GetAll(ctx context.Context) ([]country.Country, error) {
	query := `SELECT id, name FROM countries`

	logging.LogSQLQuery(*r.logger, query)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	countries := make([]country.Country, 0)
	for rows.Next() {
		var country country.Country

		err := rows.Scan(
			&country.ID,
			&country.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		countries = append(countries, country)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return countries, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*country.Country, error) {
	query := `
        SELECT id, name FROM countries
		WHERE id=$1
    `

	logging.LogSQLQuery(*r.logger, query)

	var country country.Country
	err := r.client.QueryRow(ctx, query, id).Scan(&country.ID, &country.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCountryNotFound
		}
		return nil, err
	}

	return &country, nil
}
