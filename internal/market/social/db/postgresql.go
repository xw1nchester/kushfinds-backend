package socialdb

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xw1nchester/kushfinds-backend/internal/logging"
	"github.com/xw1nchester/kushfinds-backend/internal/market/social"
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

func (r *repository) GetAll(ctx context.Context) ([]social.Social, error) {
	query := `SELECT id, name, icon FROM socials`

	logging.LogSQLQuery(r.logger, query)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	socials := make([]social.Social, 0)
	for rows.Next() {
		var social social.Social

		err := rows.Scan(
			&social.ID,
			&social.Name,
			&social.Icon,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		socials = append(socials, social)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return socials, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*social.Social, error) {
	query := `
        SELECT id, name, icon FROM socials
		WHERE id=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var social social.Social
	err := r.client.QueryRow(ctx, query, id).Scan(&social.ID, &social.Name, &social.Icon)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrSocialNotFound
		}
		return nil, err
	}

	return &social, nil
}

func (r *repository) CheckSocialsExist(ctx context.Context, IDs []int) error {
	if len(IDs) == 0 {
		return nil
	}

	placeholders := make([]string, len(IDs))
	args := make([]any, len(IDs))
	for i, id := range IDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(
		`SELECT COUNT(id) FROM socials WHERE id IN (%s)`,
		strings.Join(placeholders, ", "),
	)

	logging.LogSQLQuery(r.logger, query)

	var count int
	err := r.client.QueryRow(ctx, query, args...).Scan(&count)
	if err != nil {
		return err
	}

	if count != len(IDs) {
		return ErrSocialNotFound
	}

	return nil
}
