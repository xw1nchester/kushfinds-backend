package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vetrovegor/kushfinds-backend/internal/logging"
	"github.com/vetrovegor/kushfinds-backend/internal/market/brand"
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

func (r *repository) GetBrandsByUserID(ctx context.Context, id int) ([]brand.Brand, error) {
	return nil, nil
}

func (r *repository) GetBrandByID(ctx context.Context, id int) (*brand.Brand, error) {
	// TODO: select sub market sections, states
	query := `
		SELECT
			b.id,
			c.id,
			c.name,
			ms.id,
			ms.name,
			b.name,
			b.email,
			b.phone_number,
			b.logo,
			b.banner,
			b.created_at,
			b.updated_at
		FROM brands b
		LEFT JOIN countries c ON b.country_id = c.id
		LEFT JOIN market_sections ms ON b.market_section_id = ms.id
		WHERE b.id=$1
	`

	logging.LogSQLQuery(r.logger, query)

	var brand brand.Brand

	if err := r.client.QueryRow(ctx, query, id).Scan(
		&brand.ID,
		&brand.Country.ID,
		&brand.Country.Name,
		&brand.MarketSection.ID,
		&brand.MarketSection.Name,
		&brand.Name,
		&brand.Email,
		&brand.PhoneNumber,
		&brand.Logo,
		&brand.Banner,
		&brand.CreatedAt,
		&brand.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBrandNotFound
		}

		return nil, err
	}

	return &brand, nil
}

func (r *repository) CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error) {
	query := `
		INSERT INTO brands (user_id, country_id, market_section_id, name, email, phone_number, logo, banner)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`

	logging.LogSQLQuery(r.logger, query)

	var id int

	if err := r.client.QueryRow(
		ctx,
		query,
		data.UserID,
		data.Country.ID,
		data.MarketSection.ID,
		data.Name,
		data.Email,
		data.PhoneNumber,
		data.Logo,
		data.Banner,
	).Scan(&id); err != nil {
		return nil, err
	}

	// TODO: insert sub market sections, states WITH TRANSACTION

	return r.GetBrandByID(ctx, id)
} 