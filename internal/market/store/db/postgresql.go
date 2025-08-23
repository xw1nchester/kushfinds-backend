package storedb

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xw1nchester/kushfinds-backend/internal/logging"
	"github.com/xw1nchester/kushfinds-backend/internal/market/social"
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
			b.user_id,
			b.id,
			b.name,
			b.logo,
			s.banner,
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
			s.is_published,
			s.created_at,
			s.updated_at
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
		&store.UserID,
		&store.Brand.ID,
		&store.Brand.Name,
		&store.Brand.Logo,
		&store.Banner,
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
		&store.CreatedAt,
		&store.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrStoreNotFound
		}
		return nil, err
	}

	picsQuery := `
		SELECT url
		FROM stores_pictures
		WHERE store_id = $1
	`

	logging.LogSQLQuery(r.logger, picsQuery)

	rows, err := r.client.Query(ctx, picsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	store.Pictures = make([]string, 0)
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, err
		}
		store.Pictures = append(store.Pictures, url)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	socialsQuery := `
		SELECT s.id, s.name, s.icon, ss.url
		FROM stores_socials ss
		JOIN socials s ON ss.social_id = s.id
		WHERE ss.store_id = $1
	`

	logging.LogSQLQuery(r.logger, socialsQuery)

	rows, err = r.client.Query(ctx, socialsQuery, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	store.Socials = make([]social.EntitySocial, 0)
	for rows.Next() {
		var social social.EntitySocial
		if err := rows.Scan(
			&social.ID,
			&social.Name,
			&social.Icon,
			&social.Url,
		); err != nil {
			return nil, err
		}
		store.Socials = append(store.Socials, social)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return &store, nil
}

func (r *repository) createStoreRelatedEnitities(
	ctx context.Context,
	tx pgx.Tx,
	storeID int,
	data store.Store,
) error {
	if len(data.Pictures) > 0 {
		query := `
		INSERT INTO stores_pictures (store_id, url)
		VALUES ($1, $2)
	`
		batch := &pgx.Batch{}
		for _, url := range data.Pictures {
			logging.LogSQLQuery(r.logger, query)
			batch.Queue(query, storeID, url)
		}
		br := tx.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			return err
		}
	}

	if len(data.Socials) > 0 {
		insertSocialsQuery := `
            INSERT INTO stores_socials (store_id, social_id, url)
            VALUES ($1, $2, $3)
        `
		batch := &pgx.Batch{}
		for _, s := range data.Socials {
			logging.LogSQLQuery(r.logger, insertSocialsQuery)
			batch.Queue(insertSocialsQuery, storeID, s.ID, s.Url)
		}
		br := tx.SendBatch(ctx, batch)
		if err := br.Close(); err != nil {
			return err
		}
	}

	return nil
}

func (r *repository) CreateStore(ctx context.Context, data store.Store) (*store.Store, error) {
	tx, err := r.client.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
        INSERT INTO stores (brand_id, name, banner, description, country_id, state_id, region_id, street, house, post_code, email, phone_number, store_type_id, delivery_price, minimal_order_price, delivery_distance, is_published)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
        RETURNING id
    `

	logging.LogSQLQuery(r.logger, query)

	var id int
	if err = tx.QueryRow(
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
		data.IsPublished,
	).Scan(&id); err != nil {
		return nil, err
	}

	if len(data.Pictures) > 0 {
		if err = r.createStoreRelatedEnitities(ctx, tx, id, data); err != nil {
			return nil, err
		}
	}

	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}

	return r.GetStoreByID(ctx, id)
}

func (r *repository) GetUserStores(ctx context.Context, userID int) ([]store.StoreSummary, error) {
	query := `
		SELECT s.id, s.name, s.banner, b.id, b.name, b.logo
		FROM stores s
		LEFT JOIN brands b ON s.brand_id = b.id
		WHERE b.user_id=$1
	`

	logging.LogSQLQuery(r.logger, query)

	rows, err := r.client.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	stores := make([]store.StoreSummary, 0)
	for rows.Next() {
		var store store.StoreSummary

		err := rows.Scan(
			&store.ID,
			&store.Name,
			&store.Banner,
			&store.Brand.ID,
			&store.Brand.Name,
			&store.Brand.Logo,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %v", err)
		}

		stores = append(stores, store)
	}

	if rows.Err() != nil {
		return nil, fmt.Errorf("row error: %v", err)
	}

	return stores, nil
}
