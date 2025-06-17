package db

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

type repository struct {
	client *pgxpool.Pool
	logger *zap.Logger
}

func NewRepository(client *pgxpool.Pool, logger *zap.Logger) Repository {
	return &repository{
		client: client,
		logger: logger,
	}
}

func (r *repository) logSQLQuery(sql string) {
	r.logger.Debug("SQL query", zap.String("query", strings.Join(strings.Fields(sql), " ")))
}

// GenerateChangePassword implements code.Repository.
func (r *repository) Create(ctx context.Context, code string, codeType string, userID int, retryDate time.Time, expiryDate time.Time) error {
	sql := `
        INSERT INTO codes (code, type, user_id, retry_date, expiry_date)
        VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (type, user_id)
		DO UPDATE SET
			code = EXCLUDED.code,
			retry_date = EXCLUDED.retry_date,
			expiry_date = EXCLUDED.expiry_date;
    `

	r.logSQLQuery(sql)

	_, err := r.client.Exec(ctx, sql, code, codeType, userID, retryDate, expiryDate)

	return err
}

func (r *repository) CheckRecentlyCodeExists(ctx context.Context, codeType string, userID int) (bool, error) {
	sql := `
        SELECT id FROM codes
		WHERE type=$1 AND user_id=$2 AND retry_date>NOW()
    `

	r.logSQLQuery(sql)

	var id int
	err := r.client.QueryRow(ctx, sql, codeType, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrCodeNotFound
		}
		return false, err
	}

	return true, nil
}

func (r *repository) CheckNotExpiryCodeExists(ctx context.Context, code string, codeType string, userID int) (bool, error) {
	sql := `
        SELECT id FROM codes
		WHERE code=$1 AND type=$2 AND user_id=$3 AND expiry_date>NOW()
    `

	r.logSQLQuery(sql)

	var id int
	err := r.client.QueryRow(ctx, sql, code, codeType, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrCodeNotFound
		}
		return false, err
	}

	return true, nil
}
