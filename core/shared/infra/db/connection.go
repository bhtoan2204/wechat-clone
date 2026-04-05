package db

import (
	"context"
	"fmt"
	"time"

	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	oracle "github.com/godoes/gorm-oracle"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func NewConnection(ctx context.Context, cfg *config.Config) (*gorm.DB, error) {
	logger := logging.FromContext(ctx)
	dialector := oracle.New(oracle.Config{
		DSN: cfg.DBConfig.ConnectionURL,
	})
	db, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		logger.Errorw("open gorm oracle failed", zap.Error(err))
		return nil, stackErr.Error(fmt.Errorf("open gorm oracle failed: %v", err))
	}

	sqlDB, err := db.DB()
	if err != nil {
		logger.Errorw("get sql db failed", zap.Error(err))
		return nil, stackErr.Error(fmt.Errorf("get sql db failed: %v", err))
	}

	// Pool config
	sqlDB.SetMaxOpenConns(cfg.DBConfig.MaxOpenConnNumber)
	sqlDB.SetMaxIdleConns(cfg.DBConfig.MaxIdleConnNumber)
	sqlDB.SetConnMaxLifetime(time.Duration(cfg.DBConfig.ConnMaxLifeTimeSeconds) * time.Second)

	// Health check
	if err := sqlDB.PingContext(ctx); err != nil {
		logger.Errorw("ping db failed", zap.Error(err))
		return nil, stackErr.Error(fmt.Errorf("ping db failed: %v", err))
	}

	return db, nil
}
