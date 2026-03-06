package main

import (
	"context"
	"flag"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/shared/config"
	"go-socket/core/shared/infra/db"
	"go-socket/core/shared/pkg/logging"
	apptransport "go-socket/core/shared/transport/app"
	"os/signal"
	"syscall"
)

func main() {
	ctx, done := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	logger := logging.FromContext(ctx)
	logger.Infow("Starting application")
	defer func() {
		done()
		if r := recover(); r != nil {
			logger.Errorw("Recovered from panic", "error", r)
		}
	}()

	cfg, err := config.LoadConfig(ctx)
	if err != nil {
		logger.Errorw("Failed to load config", "error", err)
		return
	}
	appContext, err := appCtx.LoadAppCtx(ctx, cfg)
	if err != nil {
		logger.Errorw("Failed to create app context", "error", err)
		return
	}
	defer appContext.Close()

	migrateTool := db.NewMigrateTool()
	pathMigration := flag.String("path", "migration/", "path to migrations folder")
	flag.Parse()
	if err := migrateTool.Migrate(fmt.Sprintf("file://%s", *pathMigration), cfg.DBConfig.ConnectionURL); err != nil {
		logger.Errorw("Failed to migrate database", "error", err)
		return
	}

	appServer := apptransport.NewServer(cfg)
	if err := appServer.Start(ctx, appContext); err != nil {
		logger.Errorw("Failed to start app server", "error", err)
		return
	}
}
