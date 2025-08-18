package storedb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xw1nchester/kushfinds-backend/internal/logging"
	"github.com/xw1nchester/kushfinds-backend/internal/market/store"
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

func (r *repository) GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error) {
	query := `SELECT id, name FROM store_types`

	logging.LogSQLQuery(r.logger, query)

	rows, err := r.client.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	storeTypes := make([]store.StoreType, 0)
	for rows.Next() {
		var storeType store.StoreType

		err := rows.Scan(
			&storeType.ID,
			&storeType.Name,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		storeTypes = append(storeTypes, storeType)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return storeTypes, nil
}

func (r *repository) GetStoreTypeByID(ctx context.Context, id int) (*store.StoreType, error) {
	query := `
        SELECT id, name FROM store_types
		WHERE id=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var storeType store.StoreType
	err := r.client.QueryRow(ctx, query, id).Scan(&storeType.ID, &storeType.Name)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStoreTypeNotFound
		}
		return nil, err
	}

	return &storeType, nil
}

func (r *repository) GetStoreByID(ctx context.Context, id int) (*store.Store, error) {
	query := `
		SELECT
			s.id,
			b.id,
			b.name,
			b.logo,
			s.name,
			s.description,
			c.id,
			c.name,
			st.id,
			st.name,
			r.id,
			r.name,
			s.street,
			s.house,
			s.post_code,
			s.email,
			s.phone_number,
			t.id,
			t.name,
			s.delivery_price,
			s.minimal_order_price,
			s.delivery_distance,
			s.is_published
		FROM stores s
		LEFT JOIN brands b ON s.brand_id = b.id
		LEFT JOIN countries c ON s.country_id = c.id
		LEFT JOIN states st ON s.state_id = st.id
		LEFT JOIN regions r ON s.region_id = r.id
		LEFT JOIN store_types t ON s.store_type_id = t.id
		WHERE s.id=$1
	`

	logging.LogSQLQuery(r.logger, query)

	var store store.Store
	if err := r.client.QueryRow(ctx, query, id).Scan(
		&store.ID,
		&store.Brand.ID,
		&store.Brand.Name,
		&store.Brand.Logo,
		&store.Name,
		&store.Description,
		&store.Country.ID,
		&store.Country.Name,
		&store.State.ID,
		&store.State.Name,
		&store.Region.ID,
		&store.Region.Name,
		&store.Street,
		&store.House,
		&store.PostCode,
		&store.Email,
		&store.PhoneNumber,
		&store.StoreType.ID,
		&store.StoreType.Name,
		&store.DeliveryPrice,
		&store.MinimalOrderPrice,
		&store.DeliveryDistance,
		&store.IsPublished,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStoreNotFound
		}
		return nil, err
	}

	return &store, nil
}

// TODO: create pictures (сразу вынести)
func (r *repository) CreateStore(ctx context.Context, data store.Store) (*store.Store, error) {
	query := `
        INSERT INTO stores (brand_id, name, banner, description, country_id, state_id, region_id, street, house, post_code, email, phone_number, store_type_id, delivery_price, minimal_order_price, delivery_distance)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
        RETURNING id
    `

	logging.LogSQLQuery(r.logger, query)

	var id int
	if err := r.client.QueryRow(
		ctx,
		query,
		data.Brand.ID,
		data.Name,
		data.Banner,
		data.Description,
		data.Country.ID,
		data.State.ID,
		data.Region.ID,
		data.Street,
		data.House,
		data.PostCode,
		data.Email,
		data.PhoneNumber,
		data.StoreType.ID,
		data.DeliveryPrice,
		data.MinimalOrderPrice,
		data.DeliveryDistance,
	).Scan(&id); err != nil {
		return nil, err
	}

	return r.GetStoreByID(ctx, id)
}
