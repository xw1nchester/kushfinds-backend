package db

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
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

// Create implements user.Repository.
func (r *repository) Create(ctx context.Context, email string) (int, error) {
	sql := `
        INSERT INTO users (email)
        VALUES ($1)
        RETURNING id
    `

	r.logSQLQuery(sql)

	var userId int
	err := r.client.QueryRow(ctx, sql, email).Scan(&userId)
	if err != nil {
		return userId, err
	}

	return userId, nil
}

// GetByEmail implements user.Repository.
func (r *repository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	sql := `
        SELECT id, email, username, first_name, last_name, avatar, password_hash, is_verified
        FROM users
		WHERE email=$1
    `

	r.logSQLQuery(sql)

	var existingUser user.User
	if err := r.client.QueryRow(ctx, sql, email).Scan(&existingUser.ID, &existingUser.Email, &existingUser.Username, &existingUser.FirstName, &existingUser.LastName, &existingUser.Avatar, &existingUser.PasswordHash, &existingUser.IsVerified); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &existingUser, nil
}

// GetByID implements user.Repository.
func (r *repository) GetByID(ctx context.Context, id int) (user.User, error) {
	panic("unimplemented")
}
