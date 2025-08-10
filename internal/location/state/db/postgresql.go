package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state"
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

func (r *repository) GetAll(ctx context.Context) ([]state.State, error) {
	return nil, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*state.State, error) {
	query := `
        SELECT id, name FROM states
		WHERE id=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var state state.State
	err := r.client.QueryRow(ctx, query, id).Scan(&state.ID, &state.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStateNotFound
		}
		return nil, err
	}

	return &state, nil
}

// TODO: в дальнейшем реализовать GetAll, который может принимать фильтры
func (r *repository) GetAllByCountryID(ctx context.Context, countryID int) ([]state.State, error) {
	query := `SELECT id, name FROM states WHERE country_id=$1`

	logging.LogSQLQuery(r.logger, query)

	rows, err := r.client.Query(ctx, query, countryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := make([]state.State, 0)
	for rows.Next() {
		var state state.State

		err := rows.Scan(
			&state.ID,
			&state.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		states = append(states, state)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return states, nil
}

func (r *repository) CheckStatesExist(ctx context.Context, stateIDs []int) error {
	if len(stateIDs) == 0 {
        return nil
    }

    placeholders := make([]string, len(stateIDs))
    args := make([]any, len(stateIDs))
    for i, id := range stateIDs {
        placeholders[i] = fmt.Sprintf("$%d", i+1)
        args[i] = id
    }

    query := fmt.Sprintf(
        `SELECT COUNT(id) FROM states WHERE id IN (%s)`,
        strings.Join(placeholders, ", "),
    )

	logging.LogSQLQuery(r.logger, query)

    var count int
    err := r.client.QueryRow(ctx, query, args...).Scan(&count)
    if err != nil {
        return err
    }

    if count != len(stateIDs) {
        return ErrStateNotFound
    }

    return nil
}