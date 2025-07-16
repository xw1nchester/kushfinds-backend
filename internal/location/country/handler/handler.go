package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/handlers"
	"github.com/vetrovegor/kushfinds-backend/internal/location/country"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	GetAll(ctx context.Context) ([]country.Country, error)
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
	router.Route("/countries", func(countryRouter chi.Router) {
		countryRouter.Get("/", apperror.Middleware(h.GetAll))

		// TODO: states by country id
	})
}

// @Tags		location
// @Success	200		{object}	ContriesResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/countries [get]
func (h *handler) GetAll(w http.ResponseWriter, r *http.Request) error {
	countries, err := h.service.GetAll(r.Context())
	if err != nil {
		return err
	}

	render.JSON(w, r, ContriesResponse{Contries: countries})

	return nil
}
