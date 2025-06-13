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

const (
	RefreshTokenCookieName = "refresh-token"
)

type handler struct {
	service Service
	authMiddleware func(http.Handler) http.Handler
	logger  *zap.Logger
}

func NewHandler(service Service, authMiddleware func(http.Handler) http.Handler, logger *zap.Logger) handlers.Handler {
	return &handler{
		service: service,
		authMiddleware: authMiddleware,
		logger:  logger,
	}
}

func (h *handler) Register(router chi.Router) {
	router.Route("/auth", func(authRouter chi.Router) {
		authRouter.Route("/register", func(registerRouter chi.Router) {
			registerRouter.Post("/email", h.registerEmailHandler)
			registerRouter.Post("/verify", h.registerVerifyHandler)
			registerRouter.Post("/profile", h.registerProfileHandler)
			registerRouter.Post("/password", h.registerPasswordHandler)
		})
		// authRouter.Post("/register", h.registerHandler)
		// authRouter.Post("/login", h.loginHandler)
	})

	router.Group(func(priv chi.Router) {
		priv.Use(h.authMiddleware)

		priv.Get("/private", func(w http.ResponseWriter, r *http.Request) {
			h.logger.Info("hello from private route")
			render.Status(r, 200)
		})
	})
}

func (h *handler) registerEmailHandler(w http.ResponseWriter, r *http.Request) {
	var dto RegisterEmailRequest
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

	tokens, err := h.service.RegisterEmail(r.Context(), dto, r.Header.Get("User-Agent"))
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(err.Error()))
		return
	}

	cookie := &http.Cookie{
		Name:  RefreshTokenCookieName,
		Value: tokens.RefreshToken,
		// Path:     "/", // Cookie will be valid for all paths
		// Domain: "localhost", // Cookie will be valid for localhost
		// Expires:  time.Now().Add(time.Hour), // Expires in 1 hour
		// HttpOnly: true, // Prevents JavaScript access
		// Secure: false, // Set to true for HTTPS
		// SameSite: http.SameSiteLaxMode, // Recommended for most use cases
	}

	http.SetCookie(w, cookie)

	render.JSON(w, r, JwtToken{AccessToken: tokens.AccessToken})
}

func (h *handler) registerVerifyHandler(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) registerProfileHandler(w http.ResponseWriter, r *http.Request) {

}

func (h *handler) registerPasswordHandler(w http.ResponseWriter, r *http.Request) {
}

// func (h *handler) registerHandler(w http.ResponseWriter, r *http.Request) {
// 	var dto RegisterRequest
// 	err := render.DecodeJSON(r.Body, &dto)
// 	if err != nil {
// 		render.Status(r, http.StatusBadRequest)
// 		render.JSON(w, r, response.Error("failed to decode request body"))
// 		return
// 	}

// 	validate := validator.New()
// 	if err := validate.Struct(dto); err != nil {
// 		validateErr := err.(validator.ValidationErrors)
// 		render.Status(r, http.StatusBadRequest)
// 		render.JSON(w, r, response.ValidationError(validateErr))
// 		return
// 	}

// 	resp, err := h.service.Register(r.Context(), dto)
// 	if err != nil {
// 		render.Status(r, http.StatusBadRequest)
// 		render.JSON(w, r, response.Error(err.Error()))
// 		return
// 	}

// 	render.JSON(w, r, resp)
// }

// func (h *handler) loginHandler(w http.ResponseWriter, r *http.Request) {
// 	var dto LoginRequest
// 	err := render.DecodeJSON(r.Body, &dto)
// 	if err != nil {
// 		render.Status(r, http.StatusBadRequest)
// 		render.JSON(w, r, response.Error("failed to decode request body"))
// 		return
// 	}

// 	if err := validator.New().Struct(dto); err != nil {
// 		validateErr := err.(validator.ValidationErrors)
// 		render.Status(r, http.StatusBadRequest)
// 		render.JSON(w, r, response.ValidationError(validateErr))
// 		return
// 	}

// 	resp, err := h.service.Login(r.Context(), dto)
// 	if err != nil {
// 		render.Status(r, http.StatusBadRequest)
// 		render.JSON(w, r, response.Error(err.Error()))
// 		return
// 	}

// 	render.JSON(w, r, resp)
// }
