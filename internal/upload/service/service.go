package service

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"go.uber.org/zap"
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

// TODO: возможно будет работать и без fileExtension
func (s *service) UploadFile(ctx context.Context, userID int, reader io.Reader, fileExtension string) error {
	bucketName := fmt.Sprintf("user-%d", userID)
	
	exists, err := s.minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		s.logger.Error("error checking if bucket exists", zap.Error(err))
		return err
	}

	if !exists {
		err = s.minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{})
		if err != nil {
			s.logger.Error("error creating bucket", zap.Error(err))
			return err
		}
	}

	fileName := fmt.Sprintf("%s.%s", uuid.NewString(), fileExtension)

	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		s.logger.Error("error when opening file", zap.Error(err))
		return err
	}

	defer func() {
		err := f.Close()
		if err != nil {
			s.logger.Error("error when closing file", zap.Error(err))
		}

		err = os.Remove(fileName)
		if err != nil {
			s.logger.Error("error when removing file", zap.Error(err))
		}
	}()

	io.Copy(f, reader)

	ui, err := s.minioClient.FPutObject(
		ctx,
		bucketName,
		fileName,
		fileName,
		minio.PutObjectOptions{},
	)
	if err == nil {
		s.logger.Info("uploaded info",
			zap.String("bucket", ui.Bucket),
			zap.String("key", ui.Key),
			zap.String("etag", ui.ETag),
			zap.Int64("size", ui.Size),
		)
	}

	return err
}
