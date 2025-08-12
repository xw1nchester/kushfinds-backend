package service

import (
	"context"
	"io"
	"strconv"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/xw1nchester/kushfinds-backend/internal/apperror"
	"github.com/xw1nchester/kushfinds-backend/internal/upload"
	"go.uber.org/zap"
)

const (
	BucketName = "default"
)

type service struct {
	minioClient *minio.Client
	logger      *zap.Logger
}

func New(
	minioClient *minio.Client,
	logger *zap.Logger,
) *service {
	return &service{
		minioClient: minioClient,
		logger:      logger,
	}
}

func (s *service) UploadFile(ctx context.Context, reader io.Reader, size int64, contentType string) (*upload.File, error) {
	exists, err := s.minioClient.BucketExists(ctx, BucketName)
	if err != nil {
		s.logger.Error("error checking if bucket exists", zap.Error(err))
		return nil, err
	}

	if !exists {
		err = s.minioClient.MakeBucket(ctx, BucketName, minio.MakeBucketOptions{})
		if err != nil {
			s.logger.Error("error creating bucket", zap.Error(err))
			return nil, err
		}
	}

	ui, err := s.minioClient.PutObject(
		ctx,
		BucketName,
		strconv.FormatInt(time.Now().UnixMilli(), 10),
		reader,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err == nil {
		s.logger.Info("uploaded file info",
			zap.String("bucket", ui.Bucket),
			zap.String("key", ui.Key),
			zap.String("etag", ui.ETag),
			zap.Int64("size", ui.Size),
			zap.String("version_id", ui.VersionID),
			zap.Time("version_id", ui.LastModified),
		)
	}

	return &upload.File{
		Name:        ui.Key,
		ContentType: contentType,
		Size:        ui.Size,
	}, nil
}

func (s *service) GetFile(ctx context.Context, filename string) (*upload.File, error) {
	obj, err := s.minioClient.GetObject(ctx, BucketName, filename, minio.GetObjectOptions{})
	if err != nil {
		s.logger.Error("error getting object", zap.Error(err))
		return nil, apperror.ErrNotFound
	}

	stat, err := obj.Stat()
	if err != nil {
		s.logger.Error("error getting object stats", zap.Error(err))
		return nil, apperror.ErrNotFound
	}

	return &upload.File{
		Object:      obj,
		Name:        stat.Key,
		ContentType: stat.ContentType,
		Size:        stat.Size,
	}, nil
}
