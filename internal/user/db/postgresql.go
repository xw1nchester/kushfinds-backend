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

func (r *repository) Create(ctx context.Context, email string) (int, error) {
	sql := `
        INSERT INTO users (email)
        VALUES ($1)
        RETURNING id
    `

	r.logSQLQuery(sql)

	var id int
	err := r.client.QueryRow(ctx, sql, email).Scan(&id)
	if err != nil {
		return id, err
	}

	return id, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	sql := `
        SELECT id, email, username, first_name, last_name, avatar, password_hash, is_verified
        FROM users
		WHERE email=$1
    `

	r.logSQLQuery(sql)

	var existingUser user.User
	if err := r.client.QueryRow(ctx, sql, email).Scan(
		&existingUser.ID,
		&existingUser.Email,
		&existingUser.Username,
		&existingUser.FirstName,
		&existingUser.LastName,
		&existingUser.Avatar,
		&existingUser.PasswordHash,
		&existingUser.IsVerified); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &existingUser, nil
}

func (r *repository) Verify(ctx context.Context, id int) (*user.User, error) {
	sql := `
		UPDATE users
		SET is_verified=true
		WHERE id=$1
		RETURNING id, email, username, first_name, last_name, avatar, is_verified
	`

	r.logSQLQuery(sql)

	var existingUser user.User
	if err := r.client.QueryRow(ctx, sql, id).Scan(
		&existingUser.ID,
		&existingUser.Email,
		&existingUser.Username,
		&existingUser.FirstName,
		&existingUser.LastName,
		&existingUser.Avatar,
		&existingUser.IsVerified); err != nil {
		return nil, err
	}

	return &existingUser, nil
}

func (r *repository) GetByID(ctx context.Context, id int) (*user.User, error) {
	sql := `
        SELECT id, email, username, first_name, last_name, avatar, password_hash, is_verified
        FROM users
		WHERE id=$1
    `

	r.logSQLQuery(sql)

	var existingUser user.User
	if err := r.client.QueryRow(ctx, sql, id).Scan(
		&existingUser.ID,
		&existingUser.Email,
		&existingUser.Username,
		&existingUser.FirstName,
		&existingUser.LastName,
		&existingUser.Avatar,
		&existingUser.PasswordHash,
		&existingUser.IsVerified); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return &existingUser, nil
}

func (r *repository) CheckUsernameIsAvailable(ctx context.Context, username string) (bool, error) {
	sql := `
        SELECT id FROM users
		WHERE username=$1
    `

	r.logSQLQuery(sql)

	var id int
	err := r.client.QueryRow(ctx, sql, username).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, ErrUserNotFound
		}
		return false, err
	}

	return false, nil
}

func (r *repository) SetProfileInfo(ctx context.Context, data user.User) (*user.User, error) {
	sql := `
		UPDATE users
		SET username=$1, first_name=$2, last_name=$3
		WHERE id=$4
		RETURNING id, email, username, first_name, last_name, avatar, is_verified
	`

	r.logSQLQuery(sql)

	var existingUser user.User
	if err := r.client.QueryRow(ctx, sql, data.Username, data.FirstName, data.LastName, data.ID).Scan(
		&existingUser.ID,
		&existingUser.Email,
		&existingUser.Username,
		&existingUser.FirstName,
		&existingUser.LastName,
		&existingUser.Avatar,
		&existingUser.IsVerified); err != nil {
		return nil, err
	}

	return &existingUser, nil
}

func (r *repository) SetPassword(ctx context.Context, id int, passwordHash []byte) error {
	sql := `
		UPDATE users
		SET password_hash=$1
		WHERE id=$2
		RETURNING id
	`

	r.logSQLQuery(sql)

	var updatedID int
	return r.client.QueryRow(ctx, sql, passwordHash, id).Scan(&updatedID)
}
