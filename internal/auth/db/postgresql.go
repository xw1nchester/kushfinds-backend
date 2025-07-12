package db

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vetrovegor/kushfinds-backend/pkg/transactor/postgresql"
	"go.uber.org/zap"
)

var (
	ErrNotFound = errors.New("session not found")
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

func (r *repository) CreateSession(ctx context.Context, token string, userAgent string, userID int, expiryDate time.Time) error {
	query := `
        INSERT INTO sessions (token, user_agent, user_id, expiry_date)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (user_agent, user_id)
		DO UPDATE SET
			token = EXCLUDED.token,
			expiry_date = EXCLUDED.expiry_date;
    `

	r.logSQLQuery(query)

	executor := postgresql.GetExecutor(ctx, r.client)

	_, err := executor.Exec(ctx, query, token, userAgent, userID, expiryDate)

	return err
}

func (r *repository) DeleteNotExpirySessionByToken(ctx context.Context, token string) (int, error) {
	query := `
        DELETE FROM sessions
		WHERE token=$1 AND expiry_date>NOW()
		RETURNING user_id
    `

	r.logSQLQuery(query)

	executor := postgresql.GetExecutor(ctx, r.client)
	var userID int

	err := executor.QueryRow(ctx, query, token).Scan(&userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}

		return 0, err
	}

	return userID, nil
}
