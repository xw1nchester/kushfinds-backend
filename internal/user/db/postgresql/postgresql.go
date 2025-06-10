package db

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	"github.com/vetrovegor/kushfinds-backend/pkg/client/postgresql"
	"go.uber.org/zap"
)

type repository struct {
	client postgresql.Client
	logger *zap.Logger
}

func NewRepository(client postgresql.Client, logger *zap.Logger) user.Repository {
	return &repository{
		client: client,
		logger: logger,
	}
}

func (r *repository) logSQLQuery(sql string) {
	r.logger.Debug("SQL query", zap.String("query", strings.Join(strings.Fields(sql), " ")))
}

// Create implements user.Repository.
func (r *repository) Create(ctx context.Context, userData user.User) (*user.User, error) {
	sql := `
        INSERT INTO users (email, password_hash, role)
        VALUES ($1, $2, $3)
        RETURNING id, email, role
    `

	r.logSQLQuery(sql)

	// TODO: затестить scany?
	var createdUser user.User
	err := r.client.QueryRow(ctx, sql, userData.Email, userData.PasswordHash, userData.Role).Scan(&createdUser.ID, &createdUser.Email, &createdUser.Role)
	if err != nil {
		return nil, err
	}

	return &createdUser, nil
}

// GetByEmail implements user.Repository.
func (r *repository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	sql := `
        SELECT id, email, role, password_hash
        FROM users
		WHERE email=$1
    `

	r.logSQLQuery(sql)

	var existingUser user.User
	if err := r.client.QueryRow(ctx, sql, email).Scan(&existingUser.ID, &existingUser.Email, &existingUser.Role, &existingUser.PasswordHash); err != nil {
		// TODO: возвращать кастомную ошибку
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		} 

		return nil, err
	}

	return &existingUser, nil
}

// GetByID implements user.Repository.
func (r *repository) GetByID(ctx context.Context, id int) (user.User, error) {
	panic("unimplemented")
}
