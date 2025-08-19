package service

import (
	"context"
	"errors"

	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	"github.com/xw1nchester/kushfinds-backend/internal/user"
	"github.com/xw1nchester/kushfinds-backend/internal/user/db"
	"go.uber.org/zap"
)

var (
	ErrBusinessProfileNotFound = apperror.NewAppError("business profile not found")
)

type Repository interface {
	GetByID(ctx context.Context, id int) (*db.User, error)
	GetByEmail(ctx context.Context, email string) (*db.User, error)
	Create(ctx context.Context, email string) (int, error)
	Verify(ctx context.Context, id int) (*db.User, error)
	CheckUsernameIsAvailable(ctx context.Context, username string) (bool, error)
	IsAdmin(ctx context.Context, userID int) (bool, error)
	SetProfileInfo(ctx context.Context, user db.User) (*db.User, error)
	SetPassword(ctx context.Context, id int, passwordHash []byte) error
	UpdateProfile(ctx context.Context, user db.User) (*db.User, error)
	GetUserBusinessProfile(ctx context.Context, userID int) (*db.BusinessProfile, error)
	UpdateBusinessProfile(ctx context.Context, data db.BusinessProfile) (*db.BusinessProfile, error)
	CheckBusinessProfileExists(ctx context.Context, userID int) error
}

type CountryService interface {
	GetByID(ctx context.Context, id int) (*country.Country, error)
}

type StateService interface {
	GetByID(ctx context.Context, id int) (*state.State, error)
}

type RegionService interface {
	GetByID(ctx context.Context, id int) (*region.Region, error)
}

type service struct {
	repository     Repository
	countryService CountryService
	stateService   StateService
	regionService  RegionService
	logger         *zap.Logger
}

func New(
	repository Repository,
	countryService CountryService,
	stateService StateService,
	regionService RegionService,
	logger *zap.Logger,
) *service {
	return &service{
		repository:     repository,
		countryService: countryService,
		stateService:   stateService,
		regionService:  regionService,
		logger:         logger,
	}
}

// TODO: у db.User сделать метод ToDomain, а у user.User ToDB
func createUserDto(data *db.User) *user.User {
	return &user.User{
		ID:                 data.ID,
		Email:              data.Email,
		Username:           data.Username,
		FirstName:          data.FirstName,
		LastName:           data.LastName,
		Avatar:             data.Avatar,
		IsVerified:         data.IsVerified,
		PasswordHash:       data.PasswordHash,
		IsPasswordSet:      data.PasswordHash != nil,
		IsAdmin:            data.IsAdmin,
		Age:                data.Age,
		PhoneNumber:        data.PhoneNumber,
		Country:            data.Country,
		State:              data.State,
		Region:             data.Region,
		HasBusinessProfile: data.HasBusinessProfile,
	}
}

func (s *service) GetByID(ctx context.Context, id int) (*user.User, error) {
	existingUser, err := s.repository.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching user by id", zap.Error(err))

		return nil, err
	}

	return createUserDto(existingUser), nil
}

func (s *service) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	existingUser, err := s.repository.GetByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching user by email", zap.Error(err))

		return nil, err
	}

	return &user.User{
		ID:            existingUser.ID,
		Email:         existingUser.Email,
		Username:      existingUser.Username,
		FirstName:     existingUser.FirstName,
		LastName:      existingUser.LastName,
		Avatar:        existingUser.Avatar,
		IsVerified:    existingUser.IsVerified,
		PasswordHash:  existingUser.PasswordHash,
		IsPasswordSet: existingUser.PasswordHash != nil,
	}, nil
}

func (s *service) Create(ctx context.Context, email string) (int, error) {
	userID, err := s.repository.Create(ctx, email)
	if err != nil {
		s.logger.Error("unexpected error when creating user", zap.Error(err))
		return 0, err
	}

	return userID, nil
}

func (s *service) Verify(ctx context.Context, id int) (*user.User, error) {
	existingUser, err := s.repository.Verify(ctx, id)
	if err != nil {
		s.logger.Error("unexpected error when verifying user", zap.Error(err))
		return nil, err
	}

	return &user.User{
		ID:            existingUser.ID,
		Email:         existingUser.Email,
		Username:      existingUser.Username,
		FirstName:     existingUser.FirstName,
		LastName:      existingUser.LastName,
		Avatar:        existingUser.Avatar,
		IsVerified:    existingUser.IsVerified,
		PasswordHash:  existingUser.PasswordHash,
		IsPasswordSet: existingUser.PasswordHash != nil,
	}, nil
}

func (s *service) CheckUsernameIsAvailable(ctx context.Context, username string) (bool, error) {
	isAvailable, err := s.repository.CheckUsernameIsAvailable(ctx, username)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return isAvailable, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when fetching user by username", zap.Error(err))
	}

	return isAvailable, err
}

func (s *service) SetProfileInfo(ctx context.Context, data *user.User) (*user.User, error) {
	updatedUser, err := s.repository.SetProfileInfo(
		ctx,
		db.User{
			ID:        data.ID,
			Username:  data.Username,
			FirstName: data.FirstName,
			LastName:  data.LastName,
		},
	)
	if err != nil {
		s.logger.Error("unexpected error when setting user profile", zap.Error(err))
		return nil, err
	}

	return &user.User{
		ID:            updatedUser.ID,
		Email:         updatedUser.Email,
		Username:      updatedUser.Username,
		FirstName:     updatedUser.FirstName,
		LastName:      updatedUser.LastName,
		Avatar:        updatedUser.Avatar,
		IsVerified:    updatedUser.IsVerified,
		PasswordHash:  updatedUser.PasswordHash,
		IsPasswordSet: updatedUser.PasswordHash != nil,
	}, nil
}

func (s *service) SetPassword(ctx context.Context, id int, passwordHash []byte) error {
	if err := s.repository.SetPassword(ctx, id, passwordHash); err != nil {
		s.logger.Error("unexpected error when set user password", zap.Error(err))
		return err
	}

	return nil
}

func (s *service) UpdateProfile(ctx context.Context, data user.User) (*user.User, error) {
	if data.Country == nil && (data.State != nil || data.Region != nil) {
		return nil, apperror.NewAppError("country should not be empty")
	}

	if data.State == nil && data.Region != nil {
		return nil, apperror.NewAppError("state should not be empty")
	}

	if data.Country != nil {
		if _, err := s.countryService.GetByID(ctx, data.Country.ID); err != nil {
			return nil, err
		}
	}

	if data.State != nil {
		if _, err := s.stateService.GetByID(ctx, data.State.ID); err != nil {
			return nil, err
		}
	}

	if data.Region != nil {
		if _, err := s.regionService.GetByID(ctx, data.Region.ID); err != nil {
			return nil, err
		}
	}

	updatedUser, err := s.repository.UpdateProfile(
		ctx,
		db.User{
			ID:          data.ID,
			FirstName:   data.FirstName,
			LastName:    data.LastName,
			Age:         data.Age,
			PhoneNumber: data.PhoneNumber,
			Country:     data.Country,
			State:       data.State,
			Region:      data.Region,
		},
	)
	if err != nil {
		s.logger.Error("unexpected error when updating user profile", zap.Error(err))
		return nil, err
	}

	return createUserDto(updatedUser), nil
}

func (s *service) GetUserBusinessProfile(ctx context.Context, userID int) (*user.BusinessProfile, error) {
	businessProfile, err := s.repository.GetUserBusinessProfile(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrBusinessProfileNotFound) {
			return nil, nil
		}

		s.logger.Error("unexpected error when fetching business profile", zap.Error(err))
		return nil, err
	}

	return businessProfile.ToDomain(), nil
}

func (s *service) validateBusinessProfileData(ctx context.Context, data user.BusinessProfile) error {
	// TODO: check business industry id
	if _, err := s.countryService.GetByID(ctx, data.Country.ID); err != nil {
		return err
	}

	if _, err := s.stateService.GetByID(ctx, data.State.ID); err != nil {
		return err
	}

	if _, err := s.regionService.GetByID(ctx, data.Region.ID); err != nil {
		return err
	}

	return nil
}

func (s *service) UpdateBusinessProfile(ctx context.Context, data user.BusinessProfile) (*user.BusinessProfile, error) {
	if err := s.validateBusinessProfileData(ctx, data); err != nil {
		return nil, err
	}

	businessProfile, err := s.repository.UpdateBusinessProfile(
		ctx,
		db.BusinessProfile{
			UserID: data.UserID,
			BusinessIndustry: db.BusinessIndustry{
				ID: data.BusinessIndustry.ID,
			},
			BusinessName: data.BusinessName,
			Country: country.Country{
				ID: data.Country.ID,
			},
			State: state.State{
				ID: data.State.ID,
			},
			Region: region.Region{
				ID: data.Region.ID,
			},
			Email:       data.Email,
			PhoneNumber: data.PhoneNumber,
			IsVerified:  false,
		},
	)
	if err != nil {
		s.logger.Error("unexpected error when updating business profile", zap.Error(err))
		return nil, err
	}

	return businessProfile.ToDomain(), nil
}

func (s *service) CheckBusinessProfileExists(ctx context.Context, userID int) error {
	err := s.repository.CheckBusinessProfileExists(ctx, userID)
	if err != nil {
		if errors.Is(err, db.ErrBusinessProfileNotFound) {
			return ErrBusinessProfileNotFound
		}

		s.logger.Info("error when check business profile exists", zap.Error(err))

		return err
	}

	return nil
}

func (s *service) AdminUpdateBusinessProfile(
	ctx context.Context,
	adminID int,
	data user.BusinessProfile,
) (*user.BusinessProfile, error) {
	// TODO: подумать как лучше работать с is_admin в токене
	isAdmin, err := s.repository.IsAdmin(ctx, adminID)
	if err != nil {
		if errors.Is(err, db.ErrUserNotFound) {
			return nil, apperror.ErrNotFound
		}

		s.logger.Error("unexpected error when user is admin checking", zap.Error(err))

		return nil, err
	}
	if !isAdmin {
		return nil, apperror.ErrForbidden
	}

	if err := s.CheckBusinessProfileExists(ctx, data.UserID); err != nil {
		return nil, err
	}

	if err := s.validateBusinessProfileData(ctx, data); err != nil {
		return nil, err
	}

	businessProfile, err := s.repository.UpdateBusinessProfile(
		ctx,
		db.BusinessProfile{
			UserID: data.UserID,
			BusinessIndustry: db.BusinessIndustry{
				ID: data.BusinessIndustry.ID,
			},
			BusinessName: data.BusinessName,
			Country: country.Country{
				ID: data.Country.ID,
			},
			State: state.State{
				ID: data.State.ID,
			},
			Region: region.Region{
				ID: data.Region.ID,
			},
			Email:       data.Email,
			PhoneNumber: data.PhoneNumber,
			IsVerified:  data.IsVerified,
		},
	)
	if err != nil {
		s.logger.Error("unexpected error when updating business profile", zap.Error(err))
		return nil, err
	}

	return businessProfile.ToDomain(), nil
}
