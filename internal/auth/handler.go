package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/vetrovegor/kushfinds-backend/internal/handlers"
	"github.com/vetrovegor/kushfinds-backend/internal/lib/api/response"
	"go.uber.org/zap"
)

type handler struct {
	service Service
	logger  *zap.Logger
}

func NewHandler(service Service, logger *zap.Logger) handlers.Handler {
	return &handler{
		service: service,
		logger:  logger,
	}
}

func (h *handler) Register(router chi.Router) {
	router.Route("/auth", func(r chi.Router) {
		r.Post("/register", h.registerHandler)
		r.Post("/login", h.loginHandler)
	})

	router.Group(func(priv chi.Router) {
		priv.Use(AuthMiddleware)

		priv.Get("/private", func(w http.ResponseWriter, r *http.Request) {
			h.logger.Info("hello from private route")
			render.Status(r, 400)
		})
	})
}

func (h *handler) registerHandler(w http.ResponseWriter, r *http.Request) {
	var dto RegisterRequest
	err := render.DecodeJSON(r.Body, &dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error("failed to decode request body"))
		return
	}

	validate := validator.New()
	if err := validate.Struct(dto); err != nil {
		validateErr := err.(validator.ValidationErrors)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(validateErr))
		return
	}

	resp, err := h.service.Register(r.Context(), dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(err.Error()))
		return
	}

	render.JSON(w, r, resp)
}

func (h *handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	var dto LoginRequest
	err := render.DecodeJSON(r.Body, &dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error("failed to decode request body"))
		return
	}

	if err := validator.New().Struct(dto); err != nil {
		validateErr := err.(validator.ValidationErrors)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(validateErr))
		return
	}

	resp, err := h.service.Login(r.Context(), dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(err.Error()))
		return
	}

	render.JSON(w, r, resp)
}
