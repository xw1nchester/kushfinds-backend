package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/handlers"
	"github.com/xw1nchester/kushfinds-backend/internal/location/country"
	"github.com/xw1nchester/kushfinds-backend/internal/location/state"
	statehandler "github.com/xw1nchester/kushfinds-backend/internal/location/state/handler"
	"go.uber.org/zap"
)

type Service interface {
	GetAll(ctx context.Context) ([]country.Country, error)
	GetCountryStates(ctx context.Context, id int) ([]state.State, error)
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
		countryRouter.Get("/", apperror.Middleware(h.GetAllHandler))
		countryRouter.Get("/{id}/states", apperror.Middleware(h.GetCountryStatesHandler))
	})
}

// @Tags		location
// @Success	200		{object}	ContriesResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/countries [get]
func (h *handler) GetAllHandler(w http.ResponseWriter, r *http.Request) error {
	countries, err := h.service.GetAll(r.Context())
	if err != nil {
		return err
	}

	render.JSON(w, r, ContriesResponse{Contries: countries})

	return nil
}

// @Tags		location
// @Success	200			{object}	StatesResponse
// @Failure	400,500		{object}	apperror.AppError
// @Param		country_id	path		int	true	"Country ID"
// @Router		/countries/{country_id}/states [get]
func (h *handler) GetCountryStatesHandler(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return apperror.NewAppError("id should be positive integer")
	}

	states, err := h.service.GetCountryStates(r.Context(), id)
	if err != nil {
		return err
	}

	render.JSON(w, r, statehandler.StatesResponse{States: states})

	return nil
}
