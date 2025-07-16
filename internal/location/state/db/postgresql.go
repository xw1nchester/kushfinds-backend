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

func (r *repository) GetAll(ctx context.Context) ([]*State, error){
	return nil, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*State, error) {
	query := `
        SELECT id, name FROM states
		WHERE id=$1
    `

	r.logSQLQuery(query)

	var state State
	err := r.client.QueryRow(ctx, query, id).Scan(&state.ID, &state.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStateNotFound
		}
		return nil, err
	}

	return &state, nil
}