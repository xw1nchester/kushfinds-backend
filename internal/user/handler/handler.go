package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	"github.com/vetrovegor/kushfinds-backend/internal/handlers"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	GetByID(ctx context.Context, id int) (*user.User, error)
	UpdateProfile(ctx context.Context, id int, data user.User) (*user.User, error)
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
	router.Group(func(privateUserRouter chi.Router) {
		privateUserRouter.Use(h.authMiddleware)

		privateUserRouter.Get("/users/me", apperror.Middleware(h.userHandler))
		privateUserRouter.Patch("/users/profile", apperror.Middleware(h.updateProfileHandler))
	})
}

// @Tags		user
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

func (h *handler) updateProfileHandler(w http.ResponseWriter, r *http.Request) error {
	var dto user.ProfileRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		h.logger.Error(err.Error())
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	age, err := strconv.Atoi(dto.Age)
	if dto.Age != "" && (err != nil || age <= 0) {
		return apperror.NewAppError("field age should be positive int")
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	updatedUser, err := h.service.UpdateProfile(
		r.Context(),
		userID,
		user.User{
			ID:          userID,
			FirstName:   &dto.FirstName,
			LastName:    &dto.LastName,
			Age:         &age,
			PhoneNumber: &dto.PhoneNumber,
		},
	)
	if err != nil {
		return err
	}

	render.JSON(w, r, user.UserResponse{User: *updatedUser})

	return nil
}

// TODO: set avatar
