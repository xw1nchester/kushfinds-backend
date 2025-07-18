package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/vetrovegor/kushfinds-backend/internal/apperror"
	jwtauth "github.com/vetrovegor/kushfinds-backend/internal/auth/jwt"
	"github.com/vetrovegor/kushfinds-backend/internal/handlers"
	"go.uber.org/zap"
)

type Service interface {
	 UploadFile(ctx context.Context, userID int, reader io.Reader, fileExtension string) error
}

type handler struct {
	service        Service
	authMiddleware func(http.Handler) http.Handler
	logger         *zap.Logger
}

func New(
	service Service,
	authMiddleware func(http.Handler) http.Handler,
	logger *zap.Logger,
) handlers.Handler {
	return &handler{
		service:        service,
		authMiddleware: authMiddleware,
		logger:         logger,
	}
}

func (h *handler) Register(router chi.Router) {
	router.Group(func(privateRouter chi.Router) {
		privateRouter.Use(h.authMiddleware)

		privateRouter.Post("/upload", apperror.Middleware(h.uploadHandler))
	})
}

func (h *handler) uploadHandler(w http.ResponseWriter, r *http.Request) error {
	file, header, err := r.FormFile("file") //to upload use this, curl http://localhost:8080/upload -F 'file=@lifecycle.png'
	if err != nil {
		return apperror.NewAppError(fmt.Sprintf("failed to retrieving file: %s", err.Error()))
	}
	defer file.Close()

	extension := strings.Split(header.Filename, ".")[1]

	h.logger.Info("uploaded file info", zap.String("extension", extension))

	userID := r.Context().Value(jwtauth.UserIDContextKey{}).(int)

	return h.service.UploadFile(r.Context(), userID, file, extension)
}
