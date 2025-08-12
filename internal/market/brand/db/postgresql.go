package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	"github.com/xw1nchester/kushfinds-backend/internal/logging"
	"github.com/xw1nchester/kushfinds-backend/internal/market/brand"
	marketsection "github.com/xw1nchester/kushfinds-backend/internal/market/section"
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

// TODO: вовзращать только error
func (r *repository) CheckBrandNameIsAvailable(
	ctx context.Context,
	name string,
	excludeID ...int,
) (bool, error) {
	query := `
        SELECT id FROM brands
		WHERE name=$1
    `

	args := []any{name}

	if len(excludeID) > 0 {
		query += " AND id <> $2"
		args = append(args, excludeID[0])
	}

	logging.LogSQLQuery(r.logger, query)

	var id int
	err := r.client.QueryRow(ctx, query, args...).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, ErrBrandNotFound
		}
		return false, err
	}

	return false, nil
}

func (r *repository) GetUserBrands(ctx context.Context, userID int) ([]brand.BrandSummary, error) {
	query := `
		SELECT id, name, logo 
		FROM brands
		WHERE user_id=$1
	`

	logging.LogSQLQuery(r.logger, query)

	rows, err := r.client.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	brands := make([]brand.BrandSummary, 0)
	for rows.Next() {
		var brand brand.BrandSummary

		err := rows.Scan(
			&brand.ID,
			&brand.Name,
			&brand.Logo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		brands = append(brands, brand)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return brands, nil
}

func (r *repository) GetUserBrand(ctx context.Context, brandID, userID int) (*brand.Brand, error) {
	query := `
		SELECT
			b.id,
			b.user_id,
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
		WHERE b.id=$1 AND b.user_id=$2
	`

	var br brand.Brand
	if err := r.client.QueryRow(ctx, query, brandID, userID).Scan(
		&br.ID,
		&br.UserID,
		&br.Country.ID,
		&br.Country.Name,
		&br.MarketSection.ID,
		&br.MarketSection.Name,
		&br.Name,
		&br.Email,
		&br.PhoneNumber,
		&br.Logo,
		&br.Banner,
		&br.CreatedAt,
		&br.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBrandNotFound
		}
		return nil, err
	}

	statesQuery := `
		SELECT s.id, s.name
		FROM brands_states bs
		JOIN states s ON bs.state_id = s.id
		WHERE bs.brand_id = $1
	`
	rows, err := r.client.Query(ctx, statesQuery, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	br.States = make([]state.State, 0)
	for rows.Next() {
		var s state.State
		if err := rows.Scan(&s.ID, &s.Name); err != nil {
			return nil, err
		}
		br.States = append(br.States, s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	mssQuery := `
		SELECT ms.id, ms.name
		FROM brands_market_sub_sections bmss
		JOIN market_sections ms ON bmss.market_section_id = ms.id
		WHERE bmss.brand_id = $1
	`
	rows, err = r.client.Query(ctx, mssQuery, brandID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	br.MarketSubSections = make([]marketsection.MarketSection, 0)
	for rows.Next() {
		var ms marketsection.MarketSection
		if err := rows.Scan(&ms.ID, &ms.Name); err != nil {
			return nil, err
		}
		br.MarketSubSections = append(br.MarketSubSections, ms)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &br, nil
}

func (r *repository) CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error) {
	tx, err := r.client.Begin(ctx)
	if err != nil {
		return nil, err
	}
	// откатится, если не было коммита
	defer tx.Rollback(ctx)

	query := `
        INSERT INTO brands (user_id, country_id, market_section_id, name, email, phone_number, logo, banner)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
        RETURNING id
    `
	logging.LogSQLQuery(r.logger, query)

	var brandID int
	if err = tx.QueryRow(
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
	).Scan(&brandID); err != nil {
		return nil, err
	}

	if len(data.States) > 0 {
		insertStateQuery := `
            INSERT INTO brands_states (brand_id, state_id)
            VALUES ($1, $2)
        `
		batch := &pgx.Batch{}
		for _, s := range data.States {
			logging.LogSQLQuery(r.logger, insertStateQuery)
			batch.Queue(insertStateQuery, brandID, s.ID)
		}
		br := tx.SendBatch(ctx, batch)
		if err = br.Close(); err != nil {
			return nil, err
		}
	}

	if len(data.MarketSubSections) > 0 {
		insertMarketSubSectionQuery := `
            INSERT INTO brands_market_sub_sections (brand_id, market_section_id)
            VALUES ($1, $2)
        `
		batch := &pgx.Batch{}
		for _, ms := range data.MarketSubSections {
			logging.LogSQLQuery(r.logger, insertMarketSubSectionQuery)
			batch.Queue(insertMarketSubSectionQuery, brandID, ms.ID)
		}
		br := tx.SendBatch(ctx, batch)
		if err = br.Close(); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetUserBrand(ctx, brandID, data.UserID)
}

func (r *repository) CheckBrandExists(ctx context.Context, brandID, userID int) error {
	query := `
        SELECT id FROM brands
		WHERE id=$1 AND user_id=$2
    `

	logging.LogSQLQuery(r.logger, query)

	var id int
	err := r.client.QueryRow(ctx, query, brandID, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBrandNotFound
		}
		return err
	}

	return nil
}

func (r *repository) UpdateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error) {
	tx, err := r.client.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
        UPDATE brands
		SET
			country_id=$3,
			market_section_id=$4,
			name=$5,
			email=$6,
			phone_number=$7,
			logo=$8,
			banner=$9,
			updated_at=NOW()
        WHERE id=$1 AND user_id=$2
        RETURNING id
    `
	logging.LogSQLQuery(r.logger, query)

	var brandID int
	if err = tx.QueryRow(
		ctx,
		query,
		data.ID,
		data.UserID,
		data.Country.ID,
		data.MarketSection.ID,
		data.Name,
		data.Email,
		data.PhoneNumber,
		data.Logo,
		data.Banner,
	).Scan(&brandID); err != nil {
		return nil, err
	}

	deleteBrandsStatesQuery := "DELETE FROM brands_states WHERE brand_id=$1"
	logging.LogSQLQuery(r.logger, deleteBrandsStatesQuery)
	if _, err = tx.Exec(ctx, deleteBrandsStatesQuery, brandID); err != nil {
		return nil, err
	}

	deleteBrandsMarketSubSectionsQuery := "DELETE FROM brands_market_sub_sections WHERE brand_id=$1"
	logging.LogSQLQuery(r.logger, deleteBrandsMarketSubSectionsQuery)
	if _, err = tx.Exec(ctx, deleteBrandsMarketSubSectionsQuery, brandID); err != nil {
		return nil, err
	}

	if len(data.States) > 0 {
		insertStateQuery := `
            INSERT INTO brands_states (brand_id, state_id)
            VALUES ($1, $2)
        `
		batch := &pgx.Batch{}
		for _, s := range data.States {
			logging.LogSQLQuery(r.logger, insertStateQuery)
			batch.Queue(insertStateQuery, brandID, s.ID)
		}
		br := tx.SendBatch(ctx, batch)
		if err = br.Close(); err != nil {
			return nil, err
		}
	}

	if len(data.MarketSubSections) > 0 {
		insertMarketSubSectionQuery := `
            INSERT INTO brands_market_sub_sections (brand_id, market_section_id)
            VALUES ($1, $2)
        `
		batch := &pgx.Batch{}
		for _, ms := range data.MarketSubSections {
			logging.LogSQLQuery(r.logger, insertMarketSubSectionQuery)
			batch.Queue(insertMarketSubSectionQuery, brandID, ms.ID)
		}
		br := tx.SendBatch(ctx, batch)
		if err = br.Close(); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetUserBrand(ctx, brandID, data.UserID)
}

func (r *repository) DeleteBrand(ctx context.Context, brandID, userID int) error {
	query := `
        DELETE FROM brands
		WHERE id=$1 AND user_id=$2
    `

	logging.LogSQLQuery(r.logger, query)

	_, err := r.client.Exec(ctx, query, brandID, userID)

	return err
}
