package storehandler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	jwtauth "github.com/xw1nchester/kushfinds-backend/internal/auth/jwt"
	"github.com/xw1nchester/kushfinds-backend/internal/handlers"
	"github.com/xw1nchester/kushfinds-backend/internal/market/store"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error)

	CreateStore(ctx context.Context, data store.Store) (*store.Store, error)
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
	})

	// router.Route("/me/brands", func(privateBrandRouter chi.Router) {
	// 	privateBrandRouter.Use(h.authMiddleware)
	// 	privateBrandRouter.Post("/", apperror.Middleware(h.createBrandHandler))
	// 	privateBrandRouter.Get("/", apperror.Middleware(h.getUserBrandsHandler))
	// 	privateBrandRouter.Get("/{id}", apperror.Middleware(h.getUserBrandHandler))
	// 	privateBrandRouter.Patch("/{id}", apperror.Middleware(h.updateBrandHandler))
	// 	privateBrandRouter.Delete("/{id}", apperror.Middleware(h.deleteBrandHandler))
	// })
}

func (h *handler) GetAllStoreTypes(w http.ResponseWriter, r *http.Request) error {
	storeTypes, err := h.service.GetAllStoreTypes(r.Context())
	if err != nil {
		return err
	}

	render.JSON(w, r, StoreTypesResponse{StoreTypes: storeTypes})

	return nil
}

func (h *handler) createStoreHandler(w http.ResponseWriter, r *http.Request) error {
	var dto StoreRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		h.logger.Error(apperror.ErrDecodeBody.Error(), zap.Error(err))
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	createdStore, err := h.service.CreateStore(r.Context(), *dto.ToDomain(userID))
	if err != nil {
		return err
	}

	render.JSON(w, r, NewStoreResponse(*createdStore, h.staticURL))

	return nil
}
