package main

import (
	"context"
	"flag"
	"os"
	"os/signal"
	"syscall"
	"time"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/infra/db"
	"wechat-clone/core/shared/pkg/logging"

	"github.com/avast/retry-go/v4"
	"go.uber.org/zap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	logger := logging.FromContext(ctx)

	pathMigration := flag.String("path", "migration/", "path to migrations folder")
	flag.Parse()

	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		logger.Errorw("Failed to load config", zap.Error(err))
		os.Exit(1)
	}

	migrateTool := db.NewMigrateTool()
	if err := retry.Do(
		func() error {
			return migrateTool.Migrate("file://"+*pathMigration, cfg.DBConfig.ConnectionURL)
		},
		retry.Attempts(30),
		retry.Delay(4*time.Second),
		retry.DelayType(retry.FixedDelay),
		retry.LastErrorOnly(true),
		retry.OnRetry(func(n uint, err error) {
			logger.Warnw("Waiting for Oracle to become ready",
				"attempt", n+1,
				"error", err,
			)
		}),
	); err != nil {
		logger.Errorw("Failed to migrate database", zap.Error(err))
		os.Exit(1)
	}

	logger.Infow("Migrations completed successfully", "path", *pathMigration)
}
