package db

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	"github.com/xw1nchester/kushfinds-backend/internal/logging"
	"github.com/xw1nchester/kushfinds-backend/pkg/transactor/postgresql"
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

func (r *repository) GetByID(ctx context.Context, id int) (*User, error) {
	query := `
        SELECT 
			u.id, 
			u.email, 
			u.username, 
			u.first_name, 
			u.last_name, 
			u.avatar, 
			u.password_hash, 
			u.is_verified, 
			u.is_admin, 
			u.age, 
			u.phone_number,
			c.id,
			c.name,
			s.id,
			s.name,
			r.id,
			r.name,
			CASE 
				WHEN bp.user_id IS NOT NULL THEN TRUE
				ELSE FALSE
			END AS has_business_profile
        FROM users u
		LEFT JOIN countries c ON u.country_id = c.id
		LEFT JOIN states s ON u.state_id = s.id
		LEFT JOIN regions r ON u.region_id = r.id
		LEFT JOIN business_profiles bp ON u.id = bp.user_id
		WHERE u.id=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var existingUser User
	var countryID *int
	var countryName *string
	var stateID *int
	var stateName *string
	var regionID *int
	var regionName *string

	if err := r.client.QueryRow(ctx, query, id).Scan(
		&existingUser.ID,
		&existingUser.Email,
		&existingUser.Username,
		&existingUser.FirstName,
		&existingUser.LastName,
		&existingUser.Avatar,
		&existingUser.PasswordHash,
		&existingUser.IsVerified,
		&existingUser.IsAdmin,
		&existingUser.Age,
		&existingUser.PhoneNumber,
		&countryID,
		&countryName,
		&stateID,
		&stateName,
		&regionID,
		&regionName,
		&existingUser.HasBusinessProfile,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	if countryID != nil {
		existingUser.Country = &country.Country{
			ID:   *countryID,
			Name: *countryName,
		}
	}

	if stateID != nil {
		existingUser.State = &state.State{
			ID:   *stateID,
			Name: *stateName,
		}
	}

	if regionID != nil {
		existingUser.Region = &region.Region{
			ID:   *regionID,
			Name: *regionName,
		}
	}

	return &existingUser, nil
}

func (r *repository) GetByEmail(ctx context.Context, email string) (*User, error) {
	query := `
        SELECT 
			u.id, 
			u.email, 
			u.username, 
			u.first_name, 
			u.last_name, 
			u.avatar, 
			u.password_hash, 
			u.is_verified, 
			u.is_admin, 
			u.age, 
			u.phone_number,
			c.id,
			c.name,
			s.id,
			s.name,
			r.id,
			r.name
        FROM users u
		LEFT JOIN countries c ON u.country_id = c.id
		LEFT JOIN states s ON u.state_id = s.id
		LEFT JOIN regions r ON u.region_id = r.id
		WHERE u.email=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var existingUser User
	var countryID *int
	var countryName *string
	var stateID *int
	var stateName *string
	var regionID *int
	var regionName *string

	if err := r.client.QueryRow(ctx, query, email).Scan(
		&existingUser.ID,
		&existingUser.Email,
		&existingUser.Username,
		&existingUser.FirstName,
		&existingUser.LastName,
		&existingUser.Avatar,
		&existingUser.PasswordHash,
		&existingUser.IsVerified,
		&existingUser.IsAdmin,
		&existingUser.Age,
		&existingUser.PhoneNumber,
		&countryID,
		&countryName,
		&stateID,
		&stateName,
		&regionID,
		&regionName,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	if countryID != nil {
		existingUser.Country = &country.Country{
			ID:   *countryID,
			Name: *countryName,
		}
	}

	if stateID != nil {
		existingUser.State = &state.State{
			ID:   *stateID,
			Name: *stateName,
		}
	}

	if regionID != nil {
		existingUser.Region = &region.Region{
			ID:   *regionID,
			Name: *regionName,
		}
	}

	return &existingUser, nil
}

func (r *repository) Create(ctx context.Context, email string) (int, error) {
	query := `
        INSERT INTO users (email)
        VALUES ($1)
        RETURNING id
    `

	logging.LogSQLQuery(r.logger, query)

	executor := postgresql.GetExecutor(ctx, r.client)

	var id int
	if err := executor.QueryRow(ctx, query, email).Scan(&id); err != nil {
		return 0, err
	}

	return id, nil
}

func (r *repository) Verify(ctx context.Context, id int) (*User, error) {
	query := `
		UPDATE users
		SET is_verified=true
		WHERE id=$1
	`

	logging.LogSQLQuery(r.logger, query)

	executor := postgresql.GetExecutor(ctx, r.client)

	_, err := executor.Exec(ctx, query, id)
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, id)
}

func (r *repository) CheckUsernameIsAvailable(ctx context.Context, username string) (bool, error) {
	query := `
        SELECT id FROM users
		WHERE username=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var id int
	err := r.client.QueryRow(ctx, query, username).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return true, ErrUserNotFound
		}
		return false, err
	}

	return false, nil
}

func (r *repository) IsAdmin(ctx context.Context, userID int) (bool, error) {
	const query = `SELECT is_admin FROM users WHERE id = $1`

	logging.LogSQLQuery(r.logger, query)

	var isAdmin bool
	err := r.client.QueryRow(ctx, query, userID).Scan(&isAdmin)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, ErrUserNotFound
		}
		return false, err
	}

	return isAdmin, nil
}

func (r *repository) SetProfileInfo(ctx context.Context, data User) (*User, error) {
	query := `
		UPDATE users
		SET username=$1, first_name=$2, last_name=$3
		WHERE id=$4
	`

	logging.LogSQLQuery(r.logger, query)

	_, err := r.client.Exec(
		ctx,
		query,
		data.Username,
		data.FirstName,
		data.LastName,
		data.ID,
	)
	if err != nil {
		return nil, err
	}

	return r.GetByID(ctx, data.ID)
}

func (r *repository) SetPassword(ctx context.Context, id int, passwordHash []byte) error {
	query := `
		UPDATE users
		SET password_hash=$1
		WHERE id=$2
	`

	logging.LogSQLQuery(r.logger, query)

	_, err := r.client.Exec(ctx, query, passwordHash, id)

	return err
}

func (r *repository) UpdateProfile(ctx context.Context, data User) (*User, error) {
	query := `
		UPDATE users
		SET 
			first_name=$2, 
			last_name=$3, 
			age=$4, 
			phone_number=$5,
			country_id=$6,
			state_id=$7,
			region_id=$8
		WHERE id=$1
	`

	logging.LogSQLQuery(r.logger, query)

	var countryID *int
	if data.Country != nil {
		countryID = &data.Country.ID
	}

	var stateID *int
	if data.State != nil {
		stateID = &data.State.ID
	}

	var regionID *int
	if data.Region != nil {
		regionID = &data.Region.ID
	}

	if _, err := r.client.Exec(
		ctx,
		query,
		data.ID,
		data.FirstName,
		data.LastName,
		data.Age,
		data.PhoneNumber,
		countryID,
		stateID,
		regionID,
	); err != nil {
		return nil, err
	}

	return r.GetByID(ctx, data.ID)
}

func (r *repository) GetUserBusinessProfile(ctx context.Context, userID int) (*BusinessProfile, error) {
	query := `
        SELECT 
			bi.id,
			bi.name,
			bp.business_name,
			c.id,
			c.name,
			s.id,
			s.name,
			r.id,
			r.name,
			bp.email,
			bp.phone_number,
			bp.is_verified
        FROM business_profiles bp
		LEFT JOIN business_industries bi ON bp.business_industry_id = bi.id
		LEFT JOIN countries c ON bp.country_id = c.id
		LEFT JOIN states s ON bp.state_id = s.id
		LEFT JOIN regions r ON bp.region_id = r.id
		WHERE bp.user_id=$1
    `

	logging.LogSQLQuery(r.logger, query)

	var businessProfile BusinessProfile

	if err := r.client.QueryRow(ctx, query, userID).Scan(
		&businessProfile.BusinessIndustry.ID,
		&businessProfile.BusinessIndustry.Name,
		&businessProfile.BusinessName,
		&businessProfile.Country.ID,
		&businessProfile.Country.Name,
		&businessProfile.State.ID,
		&businessProfile.State.Name,
		&businessProfile.Region.ID,
		&businessProfile.Region.Name,
		&businessProfile.Email,
		&businessProfile.PhoneNumber,
		&businessProfile.IsVerified,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrBusinessProfileNotFound
		}

		return nil, err
	}

	return &businessProfile, nil
}

func (r *repository) UpdateBusinessProfile(ctx context.Context, data BusinessProfile) (*BusinessProfile, error) {
	query := `
        INSERT INTO business_profiles (user_id, business_industry_id, business_name, country_id, state_id, region_id, email, phone_number, is_verified)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (user_id)
		DO UPDATE SET
			business_industry_id = EXCLUDED.business_industry_id,
			business_name = EXCLUDED.business_name,
			country_id = EXCLUDED.country_id,
			state_id = EXCLUDED.state_id,
			region_id = EXCLUDED.region_id,
			email = EXCLUDED.email,
			phone_number = EXCLUDED.phone_number,
			is_verified = EXCLUDED.is_verified;
    `

	logging.LogSQLQuery(r.logger, query)

	if _, err := r.client.Exec(
		ctx,
		query,
		data.UserID,
		data.BusinessIndustry.ID,
		data.BusinessName,
		data.Country.ID,
		data.State.ID,
		data.Region.ID,
		data.Email,
		data.PhoneNumber,
		data.IsVerified,
	); err != nil {
		return nil, err
	}

	return r.GetUserBusinessProfile(ctx, data.UserID)
}

func (r *repository) CheckBusinessProfileExists(
	ctx context.Context,
	userID int,
	requireVerified bool,
) error {
	query := `
        SELECT user_id FROM business_profiles
		WHERE user_id=$1
    `

	if requireVerified {
		query += " AND is_verified=true"
	}

	logging.LogSQLQuery(r.logger, query)

	var id int
	err := r.client.QueryRow(ctx, query, userID).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrBusinessProfileNotFound
		}
		return err
	}

	return nil
}
