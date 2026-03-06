package app

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	notificationassembly "go-socket/core/modules/notification/assembly"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/logging"
	httptransport "go-socket/core/shared/transport/http"
)

type Server interface {
	Start(ctx context.Context, appCtx *appCtx.AppContext) error
}

type moduleServer interface {
	Start() error
	Stop() error
}

type appServer struct {
	cfg           *config.Config
	httpServer    *httptransport.Server
	moduleServers []moduleServer
}

func NewServer(cfg *config.Config) Server {
	return &appServer{
		cfg:        cfg,
		httpServer: httptransport.NewServer(cfg),
	}
}

func (s *appServer) Start(ctx context.Context, appContext *appCtx.AppContext) error {
	if err := s.buildModuleServers(appContext); err != nil {
		return err
	}

	if err := s.startModuleServers(ctx); err != nil {
		return err
	}
	defer s.stopModuleServers(ctx)

	return s.httpServer.Start(ctx, appContext)
}

func (s *appServer) buildModuleServers(appContext *appCtx.AppContext) error {
	notificationServer, err := notificationassembly.BuildServer(s.cfg, appContext)
	if err != nil {
		return fmt.Errorf("build notification server failed: %w", err)
	}

	s.moduleServers = []moduleServer{notificationServer}
	return nil
}

func (s *appServer) startModuleServers(ctx context.Context) error {
	for idx, module := range s.moduleServers {
		if err := module.Start(); err != nil {
			s.stopStartedModules(ctx, idx-1)
			return fmt.Errorf("start module server %T failed: %w", module, err)
		}
	}
	return nil
}

func (s *appServer) stopStartedModules(ctx context.Context, lastIdx int) {
	for i := lastIdx; i >= 0; i-- {
		if err := s.moduleServers[i].Stop(); err != nil {
			logging.FromContext(ctx).Errorw("Failed to stop module server", "error", err)
		}
	}
}

func (s *appServer) stopModuleServers(ctx context.Context) {
	for i := len(s.moduleServers) - 1; i >= 0; i-- {
		if err := s.moduleServers[i].Stop(); err != nil {
			logging.FromContext(ctx).Errorw("Failed to stop module server", "error", err)
		}
	}
}
