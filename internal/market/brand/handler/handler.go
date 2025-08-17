package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	jwtauth "github.com/xw1nchester/kushfinds-backend/internal/auth/jwt"
	"github.com/xw1nchester/kushfinds-backend/internal/handlers"
	"github.com/xw1nchester/kushfinds-backend/internal/market/brand"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	CreateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error)
	GetUserBrands(ctx context.Context, userID int) ([]brand.BrandSummary, error)
	GetUserBrand(ctx context.Context, brandID, userID int) (*brand.Brand, error)
	UpdateBrand(ctx context.Context, data brand.Brand) (*brand.Brand, error)
	DeleteBrand(ctx context.Context, brandID, userID int) error
}

type handler struct {
	service        Service
	authMiddleware func(http.Handler) http.Handler
	staticURL      string
	logger         *zap.Logger
}

func New(
	service Service,
	authMiddleware func(http.Handler) http.Handler,
	staticURL string,
	logger *zap.Logger,
) handlers.Handler {
	return &handler{
		service:        service,
		authMiddleware: authMiddleware,
		staticURL:      staticURL,
		logger:         logger,
	}
}

func (h *handler) Register(router chi.Router) {
	router.Route("/me/brands", func(privateBrandRouter chi.Router) {
		privateBrandRouter.Use(h.authMiddleware)
		privateBrandRouter.Post("/", apperror.Middleware(h.createBrandHandler))
		privateBrandRouter.Get("/", apperror.Middleware(h.getUserBrandsHandler))
		privateBrandRouter.Get("/{id}", apperror.Middleware(h.getUserBrandHandler))
		privateBrandRouter.Patch("/{id}", apperror.Middleware(h.updateBrandHandler))
		privateBrandRouter.Delete("/{id}", apperror.Middleware(h.deleteBrandHandler))
	})
}

// @Security	ApiKeyAuth
// @Tags		market
// @Param		request	body		BrandRequest	true	"request body"
// @Success	200		{object}	BrandResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/me/brands [post]
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

	render.JSON(w, r, NewBrandResponse(*createdBrand, h.staticURL))

	return nil
}

// @Security	ApiKeyAuth
// @Tags		market
// @Success	200		{object}	BrandsSummaryResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/me/brands [get]
func (h *handler) getUserBrandsHandler(w http.ResponseWriter, r *http.Request) error {
	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	brands, err := h.service.GetUserBrands(r.Context(), userID)
	if err != nil {
		return err
	}

	render.JSON(w, r, NewBrandsSummaryResponse(brands, h.staticURL))

	return nil
}

// @Security	ApiKeyAuth
// @Tags		market
// @Success	200		{object}	BrandResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/me/brands/{id} [get]
func (h *handler) getUserBrandHandler(w http.ResponseWriter, r *http.Request) error {
	brandID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return apperror.NewAppError("id should be positive integer")
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	brand, err := h.service.GetUserBrand(r.Context(), brandID, userID)
	if err != nil {
		return err
	}

	render.JSON(w, r, NewBrandResponse(*brand, h.staticURL))

	return nil
}

// @Security	ApiKeyAuth
// @Tags		market
// @Param		request	body		BrandRequest	true	"request body"
// @Success	200		{object}	BrandResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/me/brands/{id} [patch]
func (h *handler) updateBrandHandler(w http.ResponseWriter, r *http.Request) error {
	brandID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return apperror.NewAppError("id should be positive integer")
	}

	var dto BrandRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		h.logger.Error(apperror.ErrDecodeBody.Error(), zap.Error(err))
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)
	brandInfo := dto.ToDomain(userID)
	brandInfo.ID = brandID

	updatedBrand, err := h.service.UpdateBrand(r.Context(), *brandInfo)
	if err != nil {
		return err
	}

	render.JSON(w, r, NewBrandResponse(*updatedBrand, h.staticURL))

	return nil
}

// @Security	ApiKeyAuth
// @Tags		market
// @Success	200
// @Failure	400,500	{object}	apperror.AppError
// @Router		/me/brands/{id} [delete]
func (h *handler) deleteBrandHandler(w http.ResponseWriter, r *http.Request) error {
	brandID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return apperror.NewAppError("id should be positive integer")
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	return h.service.DeleteBrand(r.Context(), brandID, userID)
}
