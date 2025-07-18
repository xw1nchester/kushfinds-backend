package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vetrovegor/kushfinds-backend/internal/logging"
	"github.com/vetrovegor/kushfinds-backend/pkg/transactor/postgresql"
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

// GenerateChangePassword implements code.Repository.
func (r *repository) Create(ctx context.Context, code string, codeType string, userID int, retryDate time.Time, expiryDate time.Time) error {
	query := `
        INSERT INTO codes (code, type, user_id, retry_date, expiry_date)
        VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (type, user_id)
		DO UPDATE SET
			code = EXCLUDED.code,
			retry_date = EXCLUDED.retry_date,
			expiry_date = EXCLUDED.expiry_date;
    `

	logging.LogSQLQuery(r.logger, query)

	executor := postgresql.GetExecutor(ctx, r.client)

	_, err := executor.Exec(ctx, query, code, codeType, userID, retryDate, expiryDate)

	return err
}

func (r *repository) CheckRecentlyCodeExists(ctx context.Context, codeType string, userID int) (bool, error) {
	query := `
        SELECT id FROM codes
		WHERE type=$1 AND user_id=$2 AND retry_date>NOW()
    `

	logging.LogSQLQuery(r.logger, query)

	var id int
	err := r.client.QueryRow(ctx, query, codeType, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrCodeNotFound
		}
		return false, err
	}

	return true, nil
}

func (r *repository) CheckNotExpiryCodeExists(ctx context.Context, code string, codeType string, userID int) (bool, error) {
	query := `
        SELECT id FROM codes
		WHERE code=$1 AND type=$2 AND user_id=$3 AND expiry_date>NOW()
    `

	logging.LogSQLQuery(r.logger, query)

	var id int
	err := r.client.QueryRow(ctx, query, code, codeType, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrCodeNotFound
		}
		return false, err
	}

	return true, nil
}
