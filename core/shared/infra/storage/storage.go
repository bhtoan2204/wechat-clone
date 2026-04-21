package storage

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

//go:generate mockgen -package=storage -destination=storage_mock.go -source=storage.go
type Storage interface {
	PresignedGetObjectURL(ctx context.Context, objectKey string, expiry time.Duration) (string, error)
	PresignedPutObjectURL(ctx context.Context, objectKey string, expiry time.Duration) (string, time.Time, error)
}

type minioStorage struct {
	client        *minio.Client
	bucket        string
	publicBaseURL *url.URL
}

func NewMinIO(cfg config.StorageConfig) (Storage, error) {
	endpoint := strings.TrimSpace(cfg.MinIOEndpoint)
	accessKey := strings.TrimSpace(cfg.MinIOAccessKey)
	secretKey := strings.TrimSpace(cfg.MinIOSecretKey)
	bucket := strings.TrimSpace(cfg.MinIOBucket)
	publicBaseURL, err := parsePublicBaseURL(cfg.MinIOPublicBaseURL)
	if err != nil {
		return nil, stackErr.Error(err)
	}

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
		client:        client,
		bucket:        bucket,
		publicBaseURL: publicBaseURL,
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
	return s.publicURL(presignedURL), nil
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

	return s.publicURL(presignedURL), expiredAt, nil
}

func parsePublicBaseURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}

	u, err := url.Parse(raw)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("parse minio public base url failed: %w", err))
	}
	if u.Scheme == "" || u.Host == "" {
		return nil, stackErr.Error(fmt.Errorf("minio public base url must include scheme and host"))
	}

	return u, nil
}

func (s *minioStorage) publicURL(u *url.URL) string {
	if u == nil {
		return ""
	}
	if s == nil || s.publicBaseURL == nil {
		return u.String()
	}

	rewritten := *u
	rewritten.Scheme = s.publicBaseURL.Scheme
	rewritten.Host = s.publicBaseURL.Host
	return rewritten.String()
}
