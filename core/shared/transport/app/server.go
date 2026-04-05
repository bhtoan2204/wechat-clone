package app

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	ledgerassembly "go-socket/core/modules/ledger/assembly"
	notificationassembly "go-socket/core/modules/notification/assembly"
	paymentassembly "go-socket/core/modules/payment/assembly"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	httptransport "go-socket/core/shared/transport/http"

	"go.uber.org/zap"
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
	httpOptions   []httptransport.Option
	moduleServers []moduleServer
}

type Option func(*appServer)

func WithHTTPServer(server *httptransport.Server) Option {
	return func(s *appServer) {
		s.httpServer = server
	}
}

func WithHTTPModuleBuilders(builders ...httptransport.ModuleBuilder) Option {
	return func(s *appServer) {
		s.httpOptions = append(s.httpOptions, httptransport.WithModuleBuilders(builders...))
	}
}

func NewServer(cfg *config.Config, opts ...Option) Server {
	s := &appServer{
		cfg: cfg,
	}
	for _, opt := range opts {
		opt(s)
	}
	if s.httpServer == nil {
		s.httpServer = httptransport.NewServer(cfg, s.httpOptions...)
	}
	return s
}

func (s *appServer) Start(ctx context.Context, appContext *appCtx.AppContext) error {
	if err := s.buildModuleServers(appContext); err != nil {
		return stackErr.Error(err)
	}

	if err := s.startModuleServers(ctx); err != nil {
		return stackErr.Error(err)
	}
	defer s.stopModuleServers(ctx)

	return s.httpServer.Start(ctx, appContext)
}

func (s *appServer) buildModuleServers(appContext *appCtx.AppContext) error {
	notificationServer, err := notificationassembly.BuildServer(s.cfg, appContext)
	if err != nil {
		return fmt.Errorf("build notification server failed: %v", err)
	}

	ledgerServer, err := ledgerassembly.BuildServer(s.cfg, appContext)
	if err != nil {
		return fmt.Errorf("build ledger server failed: %v", err)
	}

	paymentProcessor, err := paymentassembly.BuildProcessors(s.cfg, appContext)
	if err != nil {
		return fmt.Errorf("build payment processor failed: %v", err)
	}

	s.moduleServers = []moduleServer{notificationServer, ledgerServer, paymentProcessor}
	return nil
}

func (s *appServer) startModuleServers(ctx context.Context) error {
	for idx, module := range s.moduleServers {
		if err := module.Start(); err != nil {
			s.stopStartedModules(ctx, idx-1)
			return fmt.Errorf("start module server %T failed: %v", module, err)
		}
	}
	return nil
}

func (s *appServer) stopStartedModules(ctx context.Context, lastIdx int) {
	for i := lastIdx; i >= 0; i-- {
		if err := s.moduleServers[i].Stop(); err != nil {
			logging.FromContext(ctx).Errorw("Failed to stop module server", zap.Error(err))
		}
	}
}

func (s *appServer) stopModuleServers(ctx context.Context) {
	for i := len(s.moduleServers) - 1; i >= 0; i-- {
		if err := s.moduleServers[i].Stop(); err != nil {
			logging.FromContext(ctx).Errorw("Failed to stop module server", zap.Error(err))
		}
	}
}
