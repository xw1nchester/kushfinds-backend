package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/handlers"
	"github.com/xw1nchester/kushfinds-backend/internal/upload"
	"go.uber.org/zap"
)

type Service interface {
	UploadFile(ctx context.Context, reader io.Reader, size int64, contentType string) (*upload.File, error)
	GetFile(ctx context.Context, filename string) (*upload.File, error)
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

	router.Get("/static/{filename}", apperror.Middleware(h.getFileHandler))
}

// @Security	ApiKeyAuth
// @Tags		upload
// @Accept		multipart/form-data
// @Param		file	formData	file	true	"form data"
// @Success	200		{object}	FileResponse
// @Failure	400,500	{object}	apperror.AppError
// @Router		/upload [post]
func (h *handler) uploadHandler(w http.ResponseWriter, r *http.Request) error {
	file, header, err := r.FormFile("file")
	if err != nil {
		return apperror.NewAppError(fmt.Sprintf("failed to retrieving file: %s", err.Error()))
	}
	defer file.Close()

	h.logger.Info(
		"file to upload info",
		zap.String("extension", strings.Split(header.Filename, ".")[1]),
		zap.Int64("size", header.Size),
		zap.String("content_type", header.Header.Get("Content-Type")),
	)

	dto, err := h.service.UploadFile(r.Context(), file, header.Size, header.Header.Get("Content-Type"))
	if err != nil {
		return err
	}

	render.JSON(w, r, FileResponse{File: *dto})

	return nil
}

// @Tags		upload
// @Produce	application/octet-stream
// @Param		filename	path	string	true	"file name"
// @Success	200
// @Failure	400,404,500	{object}	apperror.AppError
// @Router		/static/{filename} [get]
func (h *handler) getFileHandler(w http.ResponseWriter, r *http.Request) error {
	dto, err := h.service.GetFile(r.Context(), chi.URLParam(r, "filename"))
	if err != nil {
		return err
	}
	defer dto.Object.Close()

	// w.Header().Set("Content-Disposition", "attachment; filename="+dto.Name)
	w.Header().Set("Content-Type", dto.ContentType)
	// w.Header().Set("Content-Length", fmt.Sprintf("%d", dto.Size))

	if _, err := io.Copy(w, dto.Object); err != nil {
		h.logger.Error("failed to copy object to body", zap.Error(err))
		return err
	}

	return nil
}
