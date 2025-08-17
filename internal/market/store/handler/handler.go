package storehandler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/handlers"
	"github.com/xw1nchester/kushfinds-backend/internal/market/store"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	GetAllStoreTypes(ctx context.Context) ([]store.StoreType, error)
}

type handler struct {
	service Service
	logger  *zap.Logger
}

func New(service Service, logger *zap.Logger) handlers.Handler {
	return &handler{
		service: service,
		logger:  logger,
	}
}

func (h *handler) Register(router chi.Router) {
	router.Route("/store/types", func(storeTypeRouter chi.Router) {
		storeTypeRouter.Get("/", apperror.Middleware(h.GetAllStoreTypes))
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