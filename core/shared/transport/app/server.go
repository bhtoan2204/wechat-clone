package app

import (
	"context"
	"fmt"
	"sync"
	appCtx "wechat-clone/core/context"
	ledgerassembly "wechat-clone/core/modules/ledger/assembly"
	notificationassembly "wechat-clone/core/modules/notification/assembly"
	relationshipassembly "wechat-clone/core/modules/relationship/assembly"
	roomassembly "wechat-clone/core/modules/room/assembly"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/logging"
	baseserver "wechat-clone/core/shared/pkg/server"
	"wechat-clone/core/shared/pkg/stackErr"
	modruntime "wechat-clone/core/shared/runtime"
	httptransport "wechat-clone/core/shared/transport/http"

	"go.uber.org/zap"
)

//go:generate mockgen -package=app -destination=server_mock.go -source=server.go
type Server interface {
	Start(ctx context.Context, appCtx *appCtx.AppContext) error
	StartWithServer(ctx context.Context, appCtx *appCtx.AppContext, srv *baseserver.Server) error
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
	return s.StartWithServer(ctx, appContext, nil)
}

func (s *appServer) StartWithServer(ctx context.Context, appContext *appCtx.AppContext, srv *baseserver.Server) error {
	if err := s.buildModuleRuntimes(appContext); err != nil {
		return stackErr.Error(err)
	}

	if err := s.startModuleRuntimes(ctx); err != nil {
		return stackErr.Error(err)
	}
	defer s.stopModuleRuntimes(ctx)

	if srv != nil {
		return s.httpServer.StartWithServer(ctx, appContext, srv)
	}

	return s.httpServer.Start(ctx, appContext)
}

func (s *appServer) buildModuleRuntimes(appContext *appCtx.AppContext) error {
	notificationRuntime, err := notificationassembly.BuildMessagingRuntime(s.cfg, appContext)
	if err != nil {
		return stackErr.Error(fmt.Errorf("build notification messaging runtime failed: %w", err))
	}

	ledgerMessagingRuntime, err := ledgerassembly.BuildMessagingRuntime(s.cfg, appContext)
	if err != nil {
		return stackErr.Error(fmt.Errorf("build ledger messaging runtime failed: %w", err))
	}

	ledgerProjectionRuntime, err := ledgerassembly.BuildProjectionRuntime(s.cfg, appContext)
	if err != nil {
		return stackErr.Error(fmt.Errorf("build ledger messaging runtime failed: %w", err))
	}

	roomProjectionRuntime, err := roomassembly.BuildProjectionRuntime(s.cfg, appContext)
	if err != nil {
		return stackErr.Error(fmt.Errorf("build room projection runtime failed: %w", err))
	}

	relationshipMessagingRuntime, err := relationshipassembly.BuildMessagingRuntime(s.cfg, appContext)
	if err != nil {
		return stackErr.Error(fmt.Errorf("build relationship messaging runtime failed: %w", err))
	}

	s.moduleRuntimes = []modruntime.Module{
		notificationRuntime,
		ledgerMessagingRuntime,
		roomProjectionRuntime,
		relationshipMessagingRuntime,
		ledgerProjectionRuntime,
	}
	return nil
}

func (s *appServer) startModuleRuntimes(ctx context.Context) error {
	for idx, runtime := range s.moduleRuntimes {
		if err := runtime.Start(); err != nil {
			s.stopStartedRuntimes(ctx, idx-1)
			return stackErr.Error(fmt.Errorf("start module runtime %T failed: %w", runtime, err))
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
	var wg sync.WaitGroup

	for i := len(s.moduleRuntimes) - 1; i >= 0; i-- {
		runtime := s.moduleRuntimes[i]
		wg.Add(1)
		go func(runtime modruntime.Module) {
			defer wg.Done()
			if err := runtime.Stop(); err != nil {
				logging.FromContext(ctx).Errorw("Failed to stop module runtime", zap.Error(err))
			}
		}(runtime)
	}

	wg.Wait()
}
