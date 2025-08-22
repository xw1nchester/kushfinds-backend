package storehandler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	jwtmiddleware "github.com/xw1nchester/kushfinds-backend/internal/auth/jwt/middleware"
	"github.com/xw1nchester/kushfinds-backend/internal/handlers"
	"github.com/xw1nchester/kushfinds-backend/internal/market/store"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error)

	CreateStore(ctx context.Context, data store.Store) (*store.Store, error)
	GetUserStores(ctx context.Context, userID int) ([]store.StoreSummary, error)
	GetUserStore(ctx context.Context, storeID, userID int) (*store.Store, error)
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
	router.Route("/store/types", func(storeTypeRouter chi.Router) {
		storeTypeRouter.Get("/", apperror.Middleware(h.GetAllStoreTypes))
	})

	router.Route("/me/stores", func(privateStoreHandler chi.Router) {
		privateStoreHandler.Use(h.authMiddleware)
		privateStoreHandler.Post("/", apperror.Middleware(h.createStoreHandler))
		privateStoreHandler.Get("/", apperror.Middleware(h.getUserStoresHandler))
		privateStoreHandler.Get("/{id}", apperror.Middleware(h.getUserStoreHandler))
	})
}

// @Security	ApiKeyAuth
// @Tags		market
// @Success	200		{object}	StoreTypesResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/store/types [get]
func (h *handler) GetAllStoreTypes(w http.ResponseWriter, r *http.Request) error {
	storeTypes, err := h.service.GetAllStoreTypes(r.Context())
	if err != nil {
		return err
	}

	render.JSON(w, r, StoreTypesResponse{StoreTypes: storeTypes})

	return nil
}

// @Security	ApiKeyAuth
// @Tags		market
// @Param		request	body		StoreRequest	true	"request body"
// @Success	200		{object}	StoreResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/me/stores [post]
func (h *handler) createStoreHandler(w http.ResponseWriter, r *http.Request) error {
	var dto StoreRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		h.logger.Error(apperror.ErrDecodeBody.Error(), zap.Error(err))
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	userID := r.Context().Value(jwtmiddleware.UserIDContextKey{}).(int)

	createdStore, err := h.service.CreateStore(r.Context(), *dto.ToDomain(userID))
	if err != nil {
		return err
	}

	render.JSON(w, r, NewStoreResponse(*createdStore, h.staticURL))

	return nil
}

// @Security	ApiKeyAuth
// @Tags		market
// @Success	200		{object}	StoresSummaryResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/me/stores [get]
func (h *handler) getUserStoresHandler(w http.ResponseWriter, r *http.Request) error {
	userID := r.Context().Value(jwtmiddleware.UserIDContextKey{}).(int)

	stores, err := h.service.GetUserStores(r.Context(), userID)
	if err != nil {
		return err
	}

	render.JSON(w, r, NewStoresSummaryResponse(stores, h.staticURL))

	return nil
}

// @Security	ApiKeyAuth
// @Tags		market
// @Success	200		{object}	StoreResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/me/stores/{id} [get]
func (h *handler) getUserStoreHandler(w http.ResponseWriter, r *http.Request) error {
	storeID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return apperror.NewAppError("id should be positive integer")
	}

	userID := r.Context().Value(jwtmiddleware.UserIDContextKey{}).(int)

	store, err := h.service.GetUserStore(r.Context(), storeID, userID)
	if err != nil {
		return err
	}

	render.JSON(w, r, NewStoreResponse(*store, h.staticURL))

	return nil
}
