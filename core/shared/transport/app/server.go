package app

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	ledgerassembly "go-socket/core/modules/ledger/assembly"
	notificationassembly "go-socket/core/modules/notification/assembly"
	roomassembly "go-socket/core/modules/room/assembly"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	modruntime "go-socket/core/shared/runtime"
	httptransport "go-socket/core/shared/transport/http"

	"go.uber.org/zap"
)

//go:generate mockgen -package=app -destination=server_mock.go -source=server.go
type Server interface {
	Start(ctx context.Context, appCtx *appCtx.AppContext) error
}

type appServer struct {
	cfg            *config.Config
	httpServer     *httptransport.Server
	httpOptions    []httptransport.Option
	moduleRuntimes []modruntime.Module
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
	if err := s.buildModuleRuntimes(appContext); err != nil {
		return stackErr.Error(err)
	}

	if err := s.startModuleRuntimes(ctx); err != nil {
		return stackErr.Error(err)
	}
	defer s.stopModuleRuntimes(ctx)

	return s.httpServer.Start(ctx, appContext)
}

func (s *appServer) buildModuleRuntimes(appContext *appCtx.AppContext) error {
	notificationRuntime, err := notificationassembly.BuildMessagingRuntime(s.cfg, appContext)
	if err != nil {
		return stackErr.Error(fmt.Errorf("build notification messaging runtime failed: %v", err))
	}

	ledgerRuntime, err := ledgerassembly.BuildMessagingRuntime(s.cfg, appContext)
	if err != nil {
		return stackErr.Error(fmt.Errorf("build ledger messaging runtime failed: %v", err))
	}

	// paymentRuntime, err := paymentassembly.BuildProjectionRuntime(s.cfg, appContext)
	// if err != nil {
	// 	return stackErr.Error(fmt.Errorf("build payment projection runtime failed: %v", err))
	// }

	roomRuntime, err := roomassembly.BuildProjectionRuntime(s.cfg, appContext)
	if err != nil {
		return stackErr.Error(fmt.Errorf("build room projection runtime failed: %v", err))
	}

	s.moduleRuntimes = []modruntime.Module{
		notificationRuntime,
		ledgerRuntime,
		// paymentRuntime,
		roomRuntime,
	}
	return nil
}

func (s *appServer) startModuleRuntimes(ctx context.Context) error {
	for idx, runtime := range s.moduleRuntimes {
		if err := runtime.Start(); err != nil {
			s.stopStartedRuntimes(ctx, idx-1)
			return stackErr.Error(fmt.Errorf("start module runtime %T failed: %v", runtime, err))
		}
	}
	return nil
}

func (s *appServer) stopStartedRuntimes(ctx context.Context, lastIdx int) {
	for i := lastIdx; i >= 0; i-- {
		if err := s.moduleRuntimes[i].Stop(); err != nil {
			logging.FromContext(ctx).Errorw("Failed to stop module runtime", zap.Error(err))
		}
	}
}

func (s *appServer) stopModuleRuntimes(ctx context.Context) {
	for i := len(s.moduleRuntimes) - 1; i >= 0; i-- {
		if err := s.moduleRuntimes[i].Stop(); err != nil {
			logging.FromContext(ctx).Errorw("Failed to stop module runtime", zap.Error(err))
		}
	}
}
