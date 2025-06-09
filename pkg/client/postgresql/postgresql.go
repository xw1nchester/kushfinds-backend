package postgresql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vetrovegor/kushfinds-backend/internal/config"
)

type Client interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

func NewClient(ctx context.Context, pc config.PostgreSQL) (*pgxpool.Pool, error) {
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s", pc.Username, pc.Password, pc.Host, pc.Port, pc.Database)

	dbpool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %v", err)
	}

	if err = dbpool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %v", err)
	}

	return dbpool, nil
}
