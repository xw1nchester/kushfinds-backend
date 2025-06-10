package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/vetrovegor/kushfinds-backend/internal/handlers"
	"go.uber.org/zap"
)

type handler struct {
	service Service
	logger *zap.Logger
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
}

// TODO: вовзращать ошибки в более удобном стандартизированном виде (сделать middleware)
func (h *handler) registerHandler(w http.ResponseWriter, r *http.Request) {
	var dto RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if dto.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("email should not be empty"))
		return
	}
	
	if dto.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("password should not be empty"))
		return
	}

	resp, err := h.service.Register(r.Context(), dto)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(respBytes)
}

func (h *handler) loginHandler(w http.ResponseWriter, r *http.Request) {
	var dto LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	if dto.Email == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("email should not be empty"))
		return
	}
	
	if dto.Password == "" {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("password should not be empty"))
		return
	}

	resp, err := h.service.Login(r.Context(), dto)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(err.Error()))
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(respBytes)
}
