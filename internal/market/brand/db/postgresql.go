package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vetrovegor/kushfinds-backend/internal/location/state"
	"github.com/vetrovegor/kushfinds-backend/internal/logging"
	"github.com/vetrovegor/kushfinds-backend/internal/market/brand"
	marketsection "github.com/vetrovegor/kushfinds-backend/internal/market/section"
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

func (r *repository) CheckBrandNameIsAvailable(ctx context.Context, name string) (bool, error) {
	query := `
        SELECT id FROM brands
		WHERE name=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var id int
	err := r.client.QueryRow(ctx, query, name).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, ErrBrandNotFound
		}
		return false, err
	}

	return false, nil
}

func (r *repository) GetBrandsByUserID(ctx context.Context, id int) ([]brand.Brand, error) {
	return nil, nil
}

func (r *repository) GetBrandByID(ctx context.Context, id int) (*brand.Brand, error) {
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
		WHERE b.id=$1
	`

	var br brand.Brand
	if err := r.client.QueryRow(ctx, query, id).Scan(
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
	rows, err := r.client.Query(ctx, statesQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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
	rows, err = r.client.Query(ctx, mssQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

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

    return r.GetBrandByID(ctx, brandID)
}

