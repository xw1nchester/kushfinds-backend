package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/auth/jwt"
	"github.com/xw1nchester/kushfinds-backend/internal/handlers"
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/region"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	"github.com/xw1nchester/kushfinds-backend/internal/user"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	GetByID(ctx context.Context, id int) (*user.User, error)
	UpdateProfile(ctx context.Context, data user.User) (*user.User, error)
	GetUserBusinessProfile(ctx context.Context, userID int) (*user.BusinessProfile, error)
	UpdateBusinessProfile(ctx context.Context, data user.BusinessProfile) (*user.BusinessProfile, error)
}

type handler struct {
	service        Service
	authMiddleware func(http.Handler) http.Handler
	logger         *zap.Logger
}

func New(service Service, authMiddleware func(http.Handler) http.Handler, logger *zap.Logger) handlers.Handler {
	return &handler{
		service:        service,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

func (h *handler) Register(router chi.Router) {
	router.Route("/users", func(userRouter chi.Router) {
		userRouter.Group(func(privateUserRouter chi.Router) {
			privateUserRouter.Use(h.authMiddleware)

			privateUserRouter.Get("/me", apperror.Middleware(h.userHandler))
			privateUserRouter.Patch("/profile", apperror.Middleware(h.updateProfileHandler))

			privateUserRouter.Route("/business", func(businessRouter chi.Router) {
				businessRouter.Get("/", apperror.Middleware(h.getBusinessProfileHandler))
				businessRouter.Patch("/", apperror.Middleware(h.updateBusinessProfileHandler))
			})
		})
	})
}

// @Security	ApiKeyAuth
// @Tags		users
// @Success	200		{object}	user.UserResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/users/me [get]
func (h *handler) userHandler(w http.ResponseWriter, r *http.Request) error {
	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	existingUser, err := h.service.GetByID(r.Context(), userID)
	if err != nil {
		return err
	}

	render.JSON(w, r, user.UserResponse{User: *existingUser})

	return nil
}

// @Security	ApiKeyAuth
// @Tags		users
// @Param		request	body		ProfileRequest	true	"request body"
// @Success	200		{object}	user.UserResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/users/profile [patch]
func (h *handler) updateProfileHandler(w http.ResponseWriter, r *http.Request) error {
	var dto ProfileRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		h.logger.Error(apperror.ErrDecodeBody.Error(), zap.Error(err))
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	var countryData *country.Country
	if dto.CountryID != nil {
		countryData = &country.Country{
			ID: int(*dto.CountryID),
		}
	}

	var stateData *state.State
	if dto.StateID != nil {
		stateData = &state.State{
			ID: int(*dto.StateID),
		}
	}

	var regionData *region.Region
	if dto.RegionID != nil {
		regionData = &region.Region{
			ID: int(*dto.RegionID),
		}
	}

	updatedUser, err := h.service.UpdateProfile(
		r.Context(),
		user.User{
			ID:          userID,
			FirstName:   dto.FirstName,
			LastName:    dto.LastName,
			Age:         (*int)(dto.Age),
			PhoneNumber: dto.PhoneNumber,
			Country:     countryData,
			State:       stateData,
			Region:      regionData,
		},
	)
	if err != nil {
		return err
	}

	render.JSON(w, r, user.UserResponse{User: *updatedUser})

	return nil
}

// @Security	ApiKeyAuth
// @Tags		users
// @Success	200		{object}	user.BusinessProfileResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/users/business [get]
func (h *handler) getBusinessProfileHandler(w http.ResponseWriter, r *http.Request) error {
	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	businessProfile, err := h.service.GetUserBusinessProfile(r.Context(), userID)
	if err != nil {
		return err
	}

	render.JSON(w, r, user.BusinessProfileResponse{BusinessProfile: businessProfile})

	return nil
}

// @Security	ApiKeyAuth
// @Tags		users
// @Param		request	body		BusinessProfileRequest	true	"request body"
// @Success	200		{object}	user.BusinessProfileResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/users/business [patch]
func (h *handler) updateBusinessProfileHandler(w http.ResponseWriter, r *http.Request) error {
	var dto BusinessProfileRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		h.logger.Error(apperror.ErrDecodeBody.Error(), zap.Error(err))
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	businessProfile, err := h.service.UpdateBusinessProfile(
		r.Context(),
		user.BusinessProfile{
			UserID: userID,
			BusinessIndustry: user.BusinessIndustry{
				ID: int(dto.BusinessIndustryID),
			},
			BusinessName: dto.BusinessName,
			Country: country.Country{
				ID: int(dto.CountryID),
			},
			State: state.State{
				ID: int(dto.StateID),
			},
			Region: region.Region{
				ID: int(dto.RegionID),
			},
			Email:       dto.Email,
			PhoneNumber: dto.PhoneNumber,
		},
	)
	if err != nil {
		return err
	}

	render.JSON(w, r, user.BusinessProfileResponse{BusinessProfile: businessProfile})

	return nil
}

// TODO: set avatar
