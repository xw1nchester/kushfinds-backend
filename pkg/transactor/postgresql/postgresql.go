package postgresql

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txKey struct{}

type pgManager struct {
	db *pgxpool.Pool
}

func NewPgManager(db *pgxpool.Pool) *pgManager {
	return &pgManager{db: db}
}

func (m *pgManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := m.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}

	ctx = context.WithValue(ctx, txKey{}, tx)

	err = fn(ctx)
	if err != nil {
		if rbErr := tx.Rollback(ctx); rbErr != nil {
			return fmt.Errorf("rollback error: %v, original error: %v", rbErr, err)
		}
		return err
	}

	return tx.Commit(ctx)
}

func GetExecutor(ctx context.Context, db DBExecutor) DBExecutor {
	if tx, ok := ctx.Value(txKey{}).(pgx.Tx); ok {
		return tx
	}
	return db
}
