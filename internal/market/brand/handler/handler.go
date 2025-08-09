package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	jwtauth "github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	"github.com/vetrovegor/kushfinds-backend/internal/handlers"
	"github.com/vetrovegor/kushfinds-backend/internal/market/brand"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error)
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
	router.Route("/brands", func(brandRouter chi.Router) {
		brandRouter.Group(func(privateBrandRouter chi.Router) {
			privateBrandRouter.Use(h.authMiddleware)

			privateBrandRouter.Post("/", apperror.Middleware(h.createBrandHandler))
		})
	})
}

// @Security	ApiKeyAuth
// @Tags		market
// @Param		request	body		BrandRequest	true	"request body"
// @Success	200		{object}	BrandResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/brands [post]
func (h *handler) createBrandHandler(w http.ResponseWriter, r *http.Request) error {
	var dto BrandRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		h.logger.Error(apperror.ErrDecodeBody.Error(), zap.Error(err))
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	createdBrand, err := h.service.CreateBrand(r.Context(), *dto.ToDomain(userID))
	if err != nil {
		return err
	}

	render.JSON(w, r, BrandResponse{Brand: *createdBrand})

	return nil
}