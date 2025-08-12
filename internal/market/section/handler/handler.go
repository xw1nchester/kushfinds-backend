package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/handlers"
	marketsection "github.com/xw1nchester/kushfinds-backend/internal/market/section"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	GetAll(ctx context.Context) ([]marketsection.MarketSection, error)
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
	router.Route("/market-sections", func(marketSectionRouter chi.Router) {
		marketSectionRouter.Get("/", apperror.Middleware(h.GetAllHandler))
	})
}

// @Tags		market
// @Success	200		{object}	MarketSectionsResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/market-sections [get]
func (h *handler) GetAllHandler(w http.ResponseWriter, r *http.Request) error {
	marketSections, err := h.service.GetAll(r.Context())
	if err != nil {
		return err
	}

	render.JSON(w, r, MarketSectionsResponse{MarketSections: marketSections})

	return nil
}
