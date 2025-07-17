package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/handlers"
	"github.com/vetrovegor/kushfinds-backend/internal/location/region"
	regionhandler "github.com/vetrovegor/kushfinds-backend/internal/location/region/handler"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	GetStateRegions(ctx context.Context, id int) ([]region.Region, error)
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
	router.Route("/states", func(countryRouter chi.Router) {
		countryRouter.Get("/{id}/regions", apperror.Middleware(h.GetStateRegionsHandler))
	})
}

//	@Tags		location
//	@Success	200			{object}	RegionsResponse
//	@Failure	400,500		{object}	apperror.AppError
//	@Param		state_id	path		int	true	"State ID"
//	@Router		/states/{state_id}/regions [get]
func (h *handler) GetStateRegionsHandler(w http.ResponseWriter, r *http.Request) error {
	id, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		return apperror.NewAppError("id should be positive integer")
	}

	regions, err := h.service.GetStateRegions(r.Context(), id)
	if err != nil {
		return err
	}

	render.JSON(w, r, regionhandler.RegionsResponse{Regions: regions})

	return nil
}
