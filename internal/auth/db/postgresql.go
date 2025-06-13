package db

import (
	"context"
	"strings"
	"time"

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

func (r *repository) CreateSession(ctx context.Context, token string, userAgent string, userID int, expiryDate time.Time) error {
	sql := `
        INSERT INTO sessions (token, user_agent, user_id, expiry_date)
        VALUES ($1, $2, $3, $4)
    `

	r.logSQLQuery(sql)

	_, err := r.client.Exec(ctx, sql, token, userAgent, userID, expiryDate)
	
	return err
}
