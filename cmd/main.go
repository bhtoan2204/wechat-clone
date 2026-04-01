package main

import (
	"context"
	"flag"
	"fmt"
	appCtx "go-socket/core/context"
	accountassembly "go-socket/core/modules/account/assembly"
	ledgerassembly "go-socket/core/modules/ledger/assembly"
	notificationassembly "go-socket/core/modules/notification/assembly"
	paymentassembly "go-socket/core/modules/payment/assembly"
	roomassembly "go-socket/core/modules/room/assembly"
	"go-socket/core/shared/config"
	"go-socket/core/shared/infra/db"
	"go-socket/core/shared/pkg/logging"
	apptransport "go-socket/core/shared/transport/app"
	"go-socket/core/shared/utils"
	"os"
	"os/signal"
	"strconv"
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

	appServer := apptransport.NewServer(cfg, apptransport.WithHTTPModuleBuilders(
		accountassembly.BuildHTTPServer,
		ledgerassembly.BuildHTTPServer,
		notificationassembly.BuildHTTPServer,
		roomassembly.BuildHTTPServer,
		paymentassembly.BuildHTTPServer,
	))

	serviceName := "go-socket"
	serviceAddress, err := utils.GetInternalIP()
	if err != nil {
		logger.Errorw("Failed to detect internal IP, fallback to localhost", "error", err)
		serviceAddress = "127.0.0.1"
	}

	servicePort := cfg.ServerConfig.Port
	serviceID := fmt.Sprintf("%s-%s-%d", serviceName, serviceAddress, servicePort)
	if hostName, hostErr := os.Hostname(); hostErr == nil {
		serviceID = fmt.Sprintf("%s-%s-%d", serviceName, hostName, servicePort)
	}

	if err := appContext.GetConsulClient().RegisterService(ctx, serviceID, serviceName, serviceAddress, servicePort); err != nil {
		logger.Errorw("Failed to register service with consul",
			"serviceID", serviceID,
			"serviceName", serviceName,
			"serviceAddress", serviceAddress,
			"servicePort", servicePort,
			"error", err,
		)
		return
	}
	defer func() {
		if err := appContext.GetConsulClient().UnregisterService(ctx, serviceID); err != nil {
			logger.Errorw("Failed to unregister service from consul", "serviceID", serviceID, "error", err)
		}
	}()

	logger.Infow("Registered service with consul",
		"serviceID", serviceID,
		"serviceName", serviceName,
		"serviceAddress", serviceAddress,
		"servicePort", strconv.Itoa(servicePort),
	)

	if err := appServer.Start(ctx, appContext); err != nil {
		logger.Errorw("Failed to start app server", "error", err)
		return
	}
}
