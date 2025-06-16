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

func (h handler) Register(router chi.Router) {
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

		authRouter.Get("/refresh", h.refreshHandler)
		authRouter.Get("/logout", h.logoutHandler)
	})

	router.Group(func(priv chi.Router) {
		priv.Use(h.authMiddleware)

		priv.Get("/private", func(w http.ResponseWriter, r *http.Request) {
			h.logger.Info("hello from private route")
			render.Status(r, 200)
		})
	})
}

func (h handler) renderError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, ErrInternal) {
		render.Status(r, http.StatusInternalServerError)
		render.JSON(w, r, response.Error(ErrInternal.Error()))
	} else {
		render.Status(r, http.StatusBadRequest)
		render.JSON(w, r, response.Error(err.Error()))
	}
}

func (h handler) setRefreshTokenToCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
}

func (h handler) clearCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

//	@Tags		auth
//	@Param		request	body	EmailRequest	true	"request body"
//	@Success	200
//	@Failure	400,500	{object}	response.Response
//	@Router		/auth/register/email [post]
func (h handler) registerEmailHandler(w http.ResponseWriter, r *http.Request) {
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
		h.renderError(w, r, err)
		return
	}
}

//	@Tags		auth
//	@Param		request	body		CodeRequest	true	"request body"
//	@Success	200		{object}	AuthResponse
//	@Failure	400,500	{object}	response.Response
//	@Router		/auth/register/verify [post]
func (h handler) registerVerifyHandler(w http.ResponseWriter, r *http.Request) {
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
		h.renderError(w, r, err)
		return
	}

	h.setRefreshTokenToCookie(w, resp.RefreshToken)

	render.JSON(w, r, AuthResponse{UserResponse: resp.UserResponse, JwtToken: resp.JwtToken})
}

//	@Tags		auth
//	@Param		request	body	EmailRequest	true	"request body"
//	@Success	200
//	@Failure	400,500	{object}	response.Response
//	@Router		/auth/verify/resend [post]
func (h handler) VerifyResendHandler(w http.ResponseWriter, r *http.Request) {
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
		h.renderError(w, r, err)
		return
	}
}

//	@Security	ApiKeyAuth
//	@Tags		auth
//	@Param		request	body		ProfileRequest	true	"request body"
//	@Success	200		{object}	UserResponse
//	@Failure	400,500	{object}	response.Response
//	@Router		/auth/register/profile [patch]
func (h handler) registerProfileHandler(w http.ResponseWriter, r *http.Request) {
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
		h.renderError(w, r, err)
		return
	}

	render.JSON(w, r, user)
}

//	@Security	ApiKeyAuth
//	@Tags		auth
//	@Param		request	body	PasswordRequest	true	"request body"
//	@Success	200
//	@Failure	400,500	{object}	response.Response
//	@Router		/auth/register/password [patch]
func (h handler) registerPasswordHandler(w http.ResponseWriter, r *http.Request) {
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

//	@Tags		auth
//	@Param		request	body		EmailRequest	true	"request body"
//	@Success	200		{object}	UserResponse
//	@Failure	400,500	{object}	response.Response
//	@Router		/auth/login/email [post]
func (h handler) loginEmailHandler(w http.ResponseWriter, r *http.Request) {
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
		h.renderError(w, r, err)
		return
	}

	render.JSON(w, r, user)
}

//	@Tags		auth
//	@Param		request	body		EmailPasswordRequest	true	"request body"
//	@Success	200		{object}	AuthResponse
//	@Failure	400,500	{object}	response.Response
//	@Router		/auth/login/password [post]
func (h handler) loginPasswordHandler(w http.ResponseWriter, r *http.Request) {
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
		h.renderError(w, r, err)
		return
	}

	h.setRefreshTokenToCookie(w, resp.RefreshToken)

	render.JSON(w, r, AuthResponse{UserResponse: resp.UserResponse, JwtToken: resp.JwtToken})
}

//	@Tags		auth
//	@Success	200		{object}	JwtToken
//	@Failure	400,500	{object}	response.Response
//	@Router		/auth/refresh [get]
func (h handler) refreshHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	tokens, err := h.service.Refresh(r.Context(), cookie.Value, r.Header.Get("User-Agent"))
	if err != nil {
		h.clearCookie(w)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	h.setRefreshTokenToCookie(w, tokens.RefreshToken)

	render.JSON(w, r, JwtToken{AccessToken: tokens.AccessToken})
}

//	@Tags		auth
//	@Success	200
//	@Router		/auth/logout [get]
func (h handler) logoutHandler(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err == nil {
		h.service.Logout(r.Context(), cookie.Value)
		h.clearCookie(w)
	}
}
