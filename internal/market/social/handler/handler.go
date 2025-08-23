package socialhandler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/handlers"
	"github.com/xw1nchester/kushfinds-backend/internal/market/social"
	"go.uber.org/zap"
)

type Service interface {
	GetAll(ctx context.Context) ([]social.Social, error)
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
	router.Route("/socials", func(socialRouter chi.Router) {
		socialRouter.Get("/", apperror.Middleware(h.GetAllHandler))
	})
}

// @Tags		market
// @Success	200		{object}	SocialsResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/socials [get]
func (h *handler) GetAllHandler(w http.ResponseWriter, r *http.Request) error {
	socials, err := h.service.GetAll(r.Context())
	if err != nil {
		return err
	}

	render.JSON(w, r, SocialsResponse{Socials: socials})

	return nil
}
