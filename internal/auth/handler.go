package auth

import (
	"errors"
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

var (
	ErrDecodeBody = errors.New("failed to decode request body")
)

type handler struct {
	service        Service
	authMiddleware func(http.Handler) http.Handler
	logger         *zap.Logger
}

func NewHandler(service Service, authMiddleware func(http.Handler) http.Handler, logger *zap.Logger) handlers.Handler {
	return &handler{
		service:        service,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

func (h *handler) Register(router chi.Router) {
	router.Route("/auth", func(authRouter chi.Router) {
		authRouter.Route("/register", func(registerRouter chi.Router) {
			registerRouter.Post("/email", h.registerEmailHandler)
			registerRouter.Post("/verify", h.registerVerifyHandler)

			registerRouter.Group(func(privateRegisterRouter chi.Router) {
				privateRegisterRouter.Use(h.authMiddleware)

				privateRegisterRouter.Patch("/profile", h.registerProfileHandler)
				privateRegisterRouter.Patch("/password", h.registerPasswordHandler)
			})
		})
		
		authRouter.Route("/login", func(loginRouter chi.Router) {
			loginRouter.Post("/email", h.loginEmailHandler)
			loginRouter.Post("/password", h.loginPasswordHandler)
		})
		
		authRouter.Post("/verify/resend", h.VerifyResendHandler)

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
	var dto EmailRequest
	err := render.DecodeJSON(r.Body, &dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(ErrDecodeBody.Error()))
		return
	}

	validate := validator.New()
	if err := validate.Struct(dto); err != nil {
		validateErr := err.(validator.ValidationErrors)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(validateErr))
		return
	}

	err = h.service.RegisterEmail(r.Context(), dto)
	if err != nil {
		if errors.Is(err, ErrInternal) {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error(ErrInternal.Error()))
		} else {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(err.Error()))
		}
		return
	}
}

func (h *handler) registerVerifyHandler(w http.ResponseWriter, r *http.Request) {
	var dto CodeRequest
	err := render.DecodeJSON(r.Body, &dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(ErrDecodeBody.Error()))
		return
	}

	validate := validator.New()
	if err := validate.Struct(dto); err != nil {
		validateErr := err.(validator.ValidationErrors)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(validateErr))
		return
	}

	resp, err := h.service.RegisterVerify(r.Context(), dto, r.Header.Get("User-Agent"))
	if err != nil {
		if errors.Is(err, ErrInternal) {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error(ErrInternal.Error()))
		} else {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(err.Error()))
		}
		return
	}

	cookie := &http.Cookie{
		Name:  RefreshTokenCookieName,
		Value: resp.RefreshToken,
		// Path:     "/", // Cookie will be valid for all paths
		// Domain: "localhost", // Cookie will be valid for localhost
		// Expires:  time.Now().Add(time.Hour), // Expires in 1 hour
		// HttpOnly: true, // Prevents JavaScript access
		// Secure: false, // Set to true for HTTPS
		// SameSite: http.SameSiteLaxMode, // Recommended for most use cases
	}

	http.SetCookie(w, cookie)

	render.JSON(w, r, AuthResponse{UserResponse: resp.UserResponse, JwtToken: resp.JwtToken})
}

func (h *handler) VerifyResendHandler(w http.ResponseWriter, r *http.Request) {
	var dto EmailRequest
	err := render.DecodeJSON(r.Body, &dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(ErrDecodeBody.Error()))
		return
	}

	validate := validator.New()
	if err := validate.Struct(dto); err != nil {
		validateErr := err.(validator.ValidationErrors)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(validateErr))
		return
	}

	err = h.service.VerifyResend(r.Context(), dto)

	if err != nil {
		if errors.Is(err, ErrInternal) {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error(ErrInternal.Error()))
		} else {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(err.Error()))
		}
		return
	}
}

func (h *handler) registerProfileHandler(w http.ResponseWriter, r *http.Request) {
	var dto ProfileRequest
	err := render.DecodeJSON(r.Body, &dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(ErrDecodeBody.Error()))
		return
	}

	validate := validator.New()
	if err := validate.Struct(dto); err != nil {
		validateErr := err.(validator.ValidationErrors)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(validateErr))
		return
	}

	user, err := h.service.SaveProfileInfo(r.Context(), r.Context().Value("user_id").(int), dto)
	if err != nil {
		if errors.Is(err, ErrInternal) {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error(ErrInternal.Error()))
		} else {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(err.Error()))
		}
		return
	}

	render.JSON(w, r, user)
}

func (h *handler) registerPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var dto PasswordRequest
	err := render.DecodeJSON(r.Body, &dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(ErrDecodeBody.Error()))
		return
	}

	validate := validator.New()
	if err := validate.Struct(dto); err != nil {
		validateErr := err.(validator.ValidationErrors)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(validateErr))
		return
	}

	err = h.service.SavePassword(r.Context(), r.Context().Value("user_id").(int), dto)
	if err != nil {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, response.Error(ErrInternal.Error()))
	}
}

func (h *handler) loginEmailHandler(w http.ResponseWriter, r *http.Request) {
	var dto EmailRequest
	err := render.DecodeJSON(r.Body, &dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(ErrDecodeBody.Error()))
		return
	}

	validate := validator.New()
	if err := validate.Struct(dto); err != nil {
		validateErr := err.(validator.ValidationErrors)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(validateErr))
		return
	}

	user, err := h.service.GetUserByEmail(r.Context(), dto)
	if err != nil {
		if errors.Is(err, ErrInternal) {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error(ErrInternal.Error()))
		} else {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(err.Error()))
		}
		return
	}

	render.JSON(w, r, user)
}

func (h *handler) loginPasswordHandler(w http.ResponseWriter, r *http.Request) {
	var dto EmailPasswordRequest
	err := render.DecodeJSON(r.Body, &dto)
	if err != nil {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(ErrDecodeBody.Error()))
		return
	}

	validate := validator.New()
	if err := validate.Struct(dto); err != nil {
		validateErr := err.(validator.ValidationErrors)
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.ValidationError(validateErr))
		return
	}

	resp, err := h.service.Login(r.Context(), dto, r.Header.Get("User-Agent"))
	if err != nil {
		if errors.Is(err, ErrInternal) {
			render.Status(r, http.StatusInternalServerError)
			render.JSON(w, r, response.Error(ErrInternal.Error()))
		} else {
			render.Status(r, http.StatusBadRequest)
			render.JSON(w, r, response.Error(err.Error()))
		}
		return
	}

	cookie := &http.Cookie{
		Name:  RefreshTokenCookieName,
		Value: resp.RefreshToken,
		// Path:     "/", // Cookie will be valid for all paths
		// Domain: "localhost", // Cookie will be valid for localhost
		// Expires:  time.Now().Add(time.Hour), // Expires in 1 hour
		// HttpOnly: true, // Prevents JavaScript access
		// Secure: false, // Set to true for HTTPS
		// SameSite: http.SameSiteLaxMode, // Recommended for most use cases
	}

	http.SetCookie(w, cookie)

	render.JSON(w, r, AuthResponse{UserResponse: resp.UserResponse, JwtToken: resp.JwtToken})
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
