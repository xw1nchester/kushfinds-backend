package handler

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	"github.com/vetrovegor/kushfinds-backend/internal/auth"
	jwtauth "github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	"github.com/vetrovegor/kushfinds-backend/internal/handlers"
	"github.com/vetrovegor/kushfinds-backend/internal/user"
	"go.uber.org/zap"
)

const (
	RefreshTokenCookieName = "refresh-token"
)

var validate = validator.New()

//go:generate mockgen -source=handler.go -destination=mocks/mock.go -package=mockauthservice
type Service interface {
	RegisterEmail(ctx context.Context, dto auth.EmailRequest) error
	RegisterVerify(ctx context.Context, dto auth.CodeRequest, userAgent string) (*auth.AuthFullResponse, error)
	VerifyResend(ctx context.Context, dto auth.EmailRequest) error
	SaveProfileInfo(ctx context.Context, userID int, dto auth.ProfileRequest) (*user.UserResponse, error)
	SavePassword(ctx context.Context, userID int, dto auth.PasswordRequest) error
	GetUserByEmail(ctx context.Context, dto auth.EmailRequest) (*user.UserResponse, error)
	Login(ctx context.Context, dto auth.EmailPasswordRequest, userAgent string) (*auth.AuthFullResponse, error)
	Refresh(ctx context.Context, token string, userAgent string) (*auth.Tokens, error)
	Logout(ctx context.Context, token string) error
}

type handler struct {
	service        Service
	authMiddleware func(http.Handler) http.Handler
	logger         *zap.Logger
}

// TODO: покрыть тестами
func New(service Service, authMiddleware func(http.Handler) http.Handler, logger *zap.Logger) handlers.Handler {
	return &handler{
		service:        service,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

// TODO: покрыть тестами
func (h *handler) Register(router chi.Router) {
	router.Route("/auth", func(authRouter chi.Router) {
		authRouter.Route("/register", func(registerRouter chi.Router) {
			registerRouter.Post("/email", apperror.Middleware(h.RegisterEmailHandler))
			registerRouter.Post("/verify", apperror.Middleware(h.registerVerifyHandler))

			registerRouter.Group(func(privateRegisterRouter chi.Router) {
				privateRegisterRouter.Use(h.authMiddleware)

				privateRegisterRouter.Patch("/profile", apperror.Middleware(h.registerProfileHandler))
				privateRegisterRouter.Patch("/password", apperror.Middleware(h.registerPasswordHandler))
			})
		})

		authRouter.Route("/login", func(loginRouter chi.Router) {
			loginRouter.Post("/email", apperror.Middleware(h.loginEmailHandler))
			loginRouter.Post("/password", apperror.Middleware(h.loginPasswordHandler))
		})

		authRouter.Post("/verify/resend", apperror.Middleware(h.VerifyResendHandler))

		authRouter.Get("/refresh", apperror.Middleware(h.refreshHandler))
		authRouter.Get("/logout", apperror.Middleware(h.logoutHandler))
	})
}

func (h *handler) setRefreshTokenToCookie(w http.ResponseWriter, token string) {
	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
	}

	http.SetCookie(w, cookie)
}

func (h *handler) clearCookie(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     RefreshTokenCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	}
	http.SetCookie(w, cookie)
}

// @Tags		auth
// @Param		request	body	auth.EmailRequest	true	"request body"
// @Success	200
// @Failure	400,500	{object}	apperror.AppError
// @Router		/auth/register/email [post]
func (h *handler) RegisterEmailHandler(w http.ResponseWriter, r *http.Request) error {
	var dto auth.EmailRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	return h.service.RegisterEmail(r.Context(), dto)
}

// @Tags		auth
// @Param		request	body		auth.CodeRequest	true	"request body"
// @Success	200		{object}	auth.AuthResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/auth/register/verify [post]
func (h *handler) registerVerifyHandler(w http.ResponseWriter, r *http.Request) error {
	var dto auth.CodeRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	resp, err := h.service.RegisterVerify(r.Context(), dto, r.Header.Get("User-Agent"))
	if err != nil {
		return err
	}

	h.setRefreshTokenToCookie(w, resp.RefreshToken)

	render.JSON(w, r, auth.AuthResponse{UserResponse: resp.UserResponse, JwtToken: resp.JwtToken})

	return nil
}

// @Tags		auth
// @Param		request	body	auth.EmailRequest	true	"request body"
// @Success	200
// @Failure	400,500	{object}	apperror.AppError
// @Router		/auth/verify/resend [post]
func (h *handler) VerifyResendHandler(w http.ResponseWriter, r *http.Request) error {
	var dto auth.EmailRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	return h.service.VerifyResend(r.Context(), dto)
}

// @Security	ApiKeyAuth
// @Tags		auth
// @Param		request	body		auth.ProfileRequest	true	"request body"
// @Success	200		{object}	user.UserResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/auth/register/profile [patch]
func (h *handler) registerProfileHandler(w http.ResponseWriter, r *http.Request) error {
	var dto auth.ProfileRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	user, err := h.service.SaveProfileInfo(r.Context(), userID, dto)
	if err != nil {
		return err
	}

	render.JSON(w, r, user)

	return nil
}

// @Security	ApiKeyAuth
// @Tags		auth
// @Param		request	body	auth.PasswordRequest	true	"request body"
// @Success	200
// @Failure	400,500	{object}	apperror.AppError
// @Router		/auth/register/password [patch]
func (h *handler) registerPasswordHandler(w http.ResponseWriter, r *http.Request) error {
	var dto auth.PasswordRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	return h.service.SavePassword(r.Context(), userID, dto)
}

// @Tags		auth
// @Param		request	body		auth.EmailRequest	true	"request body"
// @Success	200		{object}	user.UserResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/auth/login/email [post]
func (h *handler) loginEmailHandler(w http.ResponseWriter, r *http.Request) error {
	var dto auth.EmailRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	user, err := h.service.GetUserByEmail(r.Context(), dto)
	if err != nil {
		return err
	}

	render.JSON(w, r, user)

	return nil
}

// @Tags		auth
// @Param		request	body		auth.EmailPasswordRequest	true	"request body"
// @Success	200		{object}	auth.AuthResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/auth/login/password [post]
func (h *handler) loginPasswordHandler(w http.ResponseWriter, r *http.Request) error {
	var dto auth.EmailPasswordRequest
	if err := render.DecodeJSON(r.Body, &dto); err != nil {
		return apperror.ErrDecodeBody
	}

	if err := validate.Struct(dto); err != nil {
		return apperror.NewValidationErr(err.(validator.ValidationErrors))
	}

	resp, err := h.service.Login(r.Context(), dto, r.Header.Get("User-Agent"))
	if err != nil {
		return err
	}

	h.setRefreshTokenToCookie(w, resp.RefreshToken)

	render.JSON(w, r, auth.AuthResponse{UserResponse: resp.UserResponse, JwtToken: resp.JwtToken})

	return nil
}

// @Tags		auth
// @Success	200		{object}	auth.JwtToken
// @Failure	400,500	{object}	apperror.AppError
// @Router		/auth/refresh [get]
func (h *handler) refreshHandler(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err != nil {
		return apperror.ErrUnauthorized
	}

	tokens, err := h.service.Refresh(r.Context(), cookie.Value, r.Header.Get("User-Agent"))
	if err != nil {
		return apperror.ErrUnauthorized
	}

	h.setRefreshTokenToCookie(w, tokens.RefreshToken)

	render.JSON(w, r, auth.JwtToken{AccessToken: tokens.AccessToken})

	return nil
}

// @Tags		auth
// @Success	200
// @Router		/auth/logout [get]
func (h *handler) logoutHandler(w http.ResponseWriter, r *http.Request) error {
	cookie, err := r.Cookie(RefreshTokenCookieName)
	if err == nil {
		h.service.Logout(r.Context(), cookie.Value)
		h.clearCookie(w)
	}

	return nil
}
