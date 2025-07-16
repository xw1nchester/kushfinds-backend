package db

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
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

func (r *repository) logSQLQuery(sql string) {
	r.logger.Debug("SQL query", zap.String("query", strings.Join(strings.Fields(sql), " ")))
}

func (r *repository) GetAll(ctx context.Context) ([]*Region, error){
	return nil, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*Region, error) {
	query := `
        SELECT id, name FROM regions
		WHERE id=$1
    `

	r.logSQLQuery(query)

	var region Region
	err := r.client.QueryRow(ctx, query, id).Scan(&region.ID, &region.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRegionNotFound
		}
		return nil, err
	}

	return &region, nil
}