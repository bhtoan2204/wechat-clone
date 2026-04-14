package storage

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

//go:generate mockgen -package=storage -destination=storage_mock.go -source=storage.go
type Storage interface {
	PresignedGetObjectURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	PresignedPutObjectURL(ctx context.Context, objectKey string, expiry time.Duration) (string, time.Time, error)
}

type minioStorage struct {
	client *minio.Client
	bucket string
}

func NewMinIO(cfg config.StorageConfig) (Storage, error) {
	endpoint := strings.TrimSpace(cfg.MinIOEndpoint)
	accessKey := strings.TrimSpace(cfg.MinIOAccessKey)
	secretKey := strings.TrimSpace(cfg.MinIOSecretKey)
	bucket := strings.TrimSpace(cfg.MinIOBucket)

	switch {
	case endpoint == "":
		return nil, stackErr.Error(fmt.Errorf("minio endpoint is required"))
	case accessKey == "":
		return nil, stackErr.Error(fmt.Errorf("minio access key is required"))
	case secretKey == "":
		return nil, stackErr.Error(fmt.Errorf("minio secret key is required"))
	case bucket == "":
		return nil, stackErr.Error(fmt.Errorf("minio bucket is required"))
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: cfg.MinIOUseSSL,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &minioStorage{
		client: client,
		bucket: bucket,
	}, nil
}

func (s *minioStorage) PresignedGetObjectURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return "", stackErr.Error(fmt.Errorf("object key is required"))
	}
	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	presignedURL, err := s.client.PresignedGetObject(ctx, s.bucket, objectKey, expiry, nil)
	if err != nil {
		return "", stackErr.Error(err)
	}
	return presignedURL.String(), nil
}

func (s *minioStorage) PresignedPutObjectURL(ctx context.Context, objectKey string, expiry time.Duration) (string, time.Time, error) {
	objectKey = strings.TrimSpace(objectKey)
	if objectKey == "" {
		return "", time.Time{}, stackErr.Error(fmt.Errorf("object key is required"))
	}

	if expiry <= 0 {
		expiry = 15 * time.Minute
	}

	expiredAt := time.Now().Add(expiry)

	presignedURL, err := s.client.PresignedPutObject(ctx, s.bucket, objectKey, expiry)
	if err != nil {
		return "", time.Time{}, stackErr.Error(err)
	}

	return presignedURL.String(), expiredAt, nil
}
