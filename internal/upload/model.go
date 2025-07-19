package upload

import (
	"github.com/minio/minio-go/v7"
)

type File struct {
	Object      *minio.Object `json:"-"`
	Name        string        `json:"name"`
	ContentType string        `json:"contentType"`
	Size        int64         `json:"size"`
}
