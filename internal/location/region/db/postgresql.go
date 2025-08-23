package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region"
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

func (r *repository) GetAll(ctx context.Context) ([]region.Region, error) {
	return nil, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*region.Region, error) {
	query := `
        SELECT id, name FROM regions
		WHERE id=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var region region.Region
	err := r.client.QueryRow(ctx, query, id).Scan(&region.ID, &region.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRegionNotFound
		}
		return nil, err
	}

	return &region, nil
}

// TODO: в дальнейшем реализовать GetAll, который может принимать фильтры
func (r *repository) GetAllByStateID(ctx context.Context, stateID int) ([]region.Region, error) {
	query := `SELECT id, name FROM regions WHERE state_id=$1`

	logging.LogSQLQuery(r.logger, query)

	rows, err := r.client.Query(ctx, query, stateID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	regions := make([]region.Region, 0)
	for rows.Next() {
		var region region.Region

		err := rows.Scan(
			&region.ID,
			&region.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		regions = append(regions, region)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return regions, nil
}

func (r *repository) CheckLocationExists(ctx context.Context, regionID, stateID, countryID int) error {
	query := `
        SELECT r.id
		FROM regions r
		JOIN states s ON r.state_id = s.id
		JOIN countries c ON s.country_id = c.id
		WHERE r.id = $1
		AND s.id = $2
		AND c.id = $3;
    `

	logging.LogSQLQuery(r.logger, query)

	var id int
	err := r.client.QueryRow(ctx, query, regionID, stateID, countryID).Scan(&id)
	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		return ErrLocationNotFound
	}

	return err
}
