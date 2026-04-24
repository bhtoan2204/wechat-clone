package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	appCtx "wechat-clone/core/context"
	accountassembly "wechat-clone/core/modules/account/assembly"
	ledgerassembly "wechat-clone/core/modules/ledger/assembly"
	notificationassembly "wechat-clone/core/modules/notification/assembly"
	paymentassembly "wechat-clone/core/modules/payment/assembly"
	relationassembly "wechat-clone/core/modules/relationship/assembly"
	roomassembly "wechat-clone/core/modules/room/assembly"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/infra/db"
	"wechat-clone/core/shared/pkg/logging"
	baseserver "wechat-clone/core/shared/pkg/server"
	apptransport "wechat-clone/core/shared/transport/app"
	"wechat-clone/core/shared/utils"

	"go.uber.org/zap"
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
		logger.Errorw("Failed to load config", zap.Error(err))
		return
	}
	appContext, err := appCtx.LoadAppCtx(ctx, cfg)
	if err != nil {
		logger.Errorw("Failed to create app context", zap.Error(err))
		return
	}
	defer appContext.Close()

	migrateTool := db.NewMigrateTool()
	pathMigration := flag.String("path", "migration/", "path to migrations folder")
	flag.Parse()
	if err := migrateTool.Migrate(fmt.Sprintf("file://%s", *pathMigration), db.DriverPostgres, cfg.DBConfig.ConnectionURL); err != nil {
		logger.Errorw("Failed to migrate database", zap.Error(err))
		return
	}

	appServer := apptransport.NewServer(cfg, apptransport.WithHTTPModuleBuilders(
		accountassembly.BuildHTTPServer,
		ledgerassembly.BuildHTTPServer,
		notificationassembly.BuildHTTPServer,
		paymentassembly.BuildHTTPServer,
		roomassembly.BuildHTTPServer,
		relationassembly.BuildHTTPServer,
	), apptransport.WithGRPCModuleBuilders(
		ledgerassembly.BuildGRPCServer,
		paymentassembly.BuildGRPCServer,
	))

	serviceName := "wechat-clone"
	serviceAddress, err := utils.GetInternalIP()
	if err != nil {
		logger.Errorw("Failed to detect internal IP, fallback to localhost", zap.Error(err))
		serviceAddress = "127.0.0.1"
	}

	httpServer, err := baseserver.New(cfg.ServerConfig.Port)
	if err != nil {
		logger.Errorw("Failed to bind HTTP server", zap.Error(err))
		return
	}

	servicePort, err := strconv.Atoi(httpServer.Port())
	if err != nil {
		logger.Errorw("Failed to parse bound HTTP port", zap.String("port", httpServer.Port()), zap.Error(err))
		return
	}

	serviceID := fmt.Sprintf("%s-%s-%d", serviceName, serviceAddress, servicePort)
	if hostName, hostErr := os.Hostname(); hostErr == nil {
		serviceID = fmt.Sprintf("%s-%s-%d", serviceName, hostName, servicePort)
	}

	if cfg.ServerConfig.Environment == "production" {
		if err := appContext.GetConsulClient().RegisterService(ctx, serviceID, serviceName, serviceAddress, servicePort); err != nil {
			logger.Errorw("Failed to register service with consul",
				zap.String("serviceID", serviceID),
				zap.String("serviceName", serviceName),
				zap.String("serviceAddress", serviceAddress),
				zap.Int("servicePort", servicePort),
				zap.Error(err),
			)
			return
		}
		defer func() {
			if err := appContext.GetConsulClient().UnregisterService(ctx, serviceID); err != nil {
				logger.Errorw("Failed to unregister service from consul", "serviceID", serviceID, zap.Error(err))
			}
		}()
	}

	logger.Infow("Registered service with consul",
		"serviceID", serviceID,
		"serviceName", serviceName,
		"serviceAddress", serviceAddress,
		"servicePort", strconv.Itoa(servicePort),
	)

	if err := appServer.StartWithServer(ctx, appContext, httpServer); err != nil {
		logger.Errorw("Failed to start app server", zap.Error(err))
		return
	}
}
