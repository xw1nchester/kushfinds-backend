package handler

import "github.com/vetrovegor/kushfinds-backend/internal/upload"

type FileResponse struct {
	File upload.File `json:"file"`
}