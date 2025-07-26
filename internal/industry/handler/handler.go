package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/handlers"
	"github.com/vetrovegor/kushfinds-backend/internal/industry"
	"go.uber.org/zap"
)

var validate = validator.New()

type Service interface {
	GetAll(ctx context.Context) ([]industry.Industry, error)
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
	router.Route("/industries", func(industryRouter chi.Router) {
		industryRouter.Get("/", apperror.Middleware(h.GetAllHandler))
	})
}

//	@Tags		industry
//	@Success	200		{object}	IndustriesResponse
//	@Failure	400,500	{object}	apperror.AppError
//	@Router		/industries [get]
func (h *handler) GetAllHandler(w http.ResponseWriter, r *http.Request) error {
	industries, err := h.service.GetAll(r.Context())
	if err != nil {
		return err
	}

	render.JSON(w, r, IndustriesResponse{Industries: industries})

	return nil
}
