package db

import (
	"context"
	"errors"
	"fmt"
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

func (r *repository) GetAll(ctx context.Context) ([]Country, error){
	query := `SELECT id, name FROM countries`

	r.logSQLQuery(query)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var countries []Country
	for rows.Next() {
		var country Country
		
		err := rows.Scan(
			&country.ID,
			&country.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("row scanning error: %v", err)
		}

		countries = append(countries, country)
	}

	return countries, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*Country, error) {
	query := `
        SELECT id, name FROM countries
		WHERE id=$1
    `

	r.logSQLQuery(query)

	var country Country
	err := r.client.QueryRow(ctx, query, id).Scan(&country.ID, &country.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCountryNotFound
		}
		return nil, err
	}

	return &country, nil
}