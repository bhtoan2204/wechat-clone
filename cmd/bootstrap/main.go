package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"wechat-clone/core/shared/config"
	cassandraclient "wechat-clone/core/shared/infra/cassandra"
	"wechat-clone/core/shared/pkg/logging"

	"github.com/avast/retry-go/v4"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := logging.FromContext(ctx)

	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		logger.Errorw("Failed to load config", zap.Error(err))
		os.Exit(1)
	}

	if err := ensureCassandra(ctx, cfg); err != nil {
		logger.Errorw("Failed to prepare Cassandra", zap.Error(err))
		os.Exit(1)
	}

	if err := ensureMinIOBucket(ctx, cfg); err != nil {
		logger.Errorw("Failed to prepare MinIO bucket", zap.Error(err))
		os.Exit(1)
	}

	logger.Infow("Bootstrap completed successfully")
}

func ensureCassandra(ctx context.Context, cfg *config.Config) error {
	if cfg == nil || !cfg.CassandraConfig.Enabled {
		return nil
	}

	return retry.Do(
		func() error {
			select {
			case <-ctx.Done():
				return retry.Unrecoverable(ctx.Err())
			default:
			}
			session, err := cassandraclient.NewSession(ctx, cfg.CassandraConfig)
			if err != nil {
				return err
			}
			if session != nil {
				session.Close()
			}
			return nil
		},
		retry.Attempts(20),
		retry.Delay(3*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			logging.FromContext(ctx).Warnw("Waiting for Cassandra to become ready",
				"attempt", n+1,
				"error", err,
			)
		}),
	)
}

func ensureMinIOBucket(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("config is required")
	}

	endpoint := strings.TrimSpace(cfg.StorageConfig.MinIOEndpoint)
	accessKey := strings.TrimSpace(cfg.StorageConfig.MinIOAccessKey)
	secretKey := strings.TrimSpace(cfg.StorageConfig.MinIOSecretKey)
	bucket := strings.TrimSpace(cfg.StorageConfig.MinIOBucket)

	if endpoint == "" || accessKey == "" || secretKey == "" || bucket == "" {
		return fmt.Errorf("minio endpoint, access key, secret key, and bucket are required")
	}

	return retry.Do(
		func() error {
			client, err := minio.New(endpoint, &minio.Options{
				Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
				Secure: cfg.StorageConfig.MinIOUseSSL,
			})
			if err != nil {
				return err
			}

			exists, err := client.BucketExists(ctx, bucket)
			if err != nil {
				return err
			}
			if exists {
				return nil
			}

			return client.MakeBucket(ctx, bucket, minio.MakeBucketOptions{})
		},
		retry.Attempts(20),
		retry.Delay(3*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			logging.FromContext(ctx).Warnw("Waiting for MinIO to become ready",
				"attempt", n+1,
				"error", err,
			)
		}),
	)
}
