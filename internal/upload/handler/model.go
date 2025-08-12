package handler

import "github.com/xw1nchester/kushfinds-backend/internal/upload"

type FileResponse struct {
	File upload.File `json:"file"`
}
