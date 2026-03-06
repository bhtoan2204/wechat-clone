package http

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	accountassembly "go-socket/core/modules/account/assembly"
	roomassembly "go-socket/core/modules/room/assembly"
	"go-socket/core/shared/config"
	"go-socket/core/shared/constant"
	"go-socket/core/shared/infra/idempotency"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/server"
	"go-socket/core/shared/transport/http/middleware"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

var _ App = (*Server)(nil)

type App interface {
	Routes(ctx context.Context, appCtx *appCtx.AppContext) *gin.Engine
	Start(ctx context.Context, appCtx *appCtx.AppContext) error
}

type moduleServer interface {
	RegisterPublicRoutes(routes *gin.RouterGroup)
	RegisterPrivateRoutes(routes *gin.RouterGroup)
	Stop(ctx context.Context) error
}

type Server struct {
	cfg           *config.Config
	router        *gin.Engine
	httpServer    *http.Server
	moduleServers []moduleServer
	appCtx        *appCtx.AppContext
}

func NewServer(cfg *config.Config) *Server {
	return &Server{
		cfg: cfg,
	}
}

func (s *Server) Routes(ctx context.Context, appCtx *appCtx.AppContext) *gin.Engine {
	r := gin.New()
	r.MaxMultipartMemory = 50 << 20
	r.RedirectTrailingSlash = false
	cache := appCtx.GetCache()
	r.Use(middleware.SetRequestID())
	idemStore := idempotency.NewRedisStore(cache)
	idemManager := idempotency.NewManager(
		idemStore,
		constant.DEFAULT_IDEMPOTENCY_LOCK_TTL,
		constant.DEFAULT_IDEMPOTENCY_DONE_TTL,
	)
	r.Use(middleware.IdempotencyMiddleware(idemManager))
	r.Use(middleware.RateLimitMiddleware(cache))
	r.Use(gin.CustomRecovery(func(c *gin.Context, err interface{}) {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": gin.H{"error": "something went wrong"}})
	}))

	// cors
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{
		"*",
		"Origin",
		"Content-Length",
		"Content-Type",
		"Authorization",
		"X-Inside-Token",
	}
	r.Use(cors.New(corsConfig))

	pingHandler := func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"clientIP": ctx.ClientIP(),
			},
		})
	}
	r.GET("/health-check", pingHandler)
	r.HEAD("/health-check", pingHandler)

	s.router = r
	s.appCtx = appCtx

	// public api
	s.registerPublicAPI()
	s.registerPrivateAPI()
	return r
}

func (s *Server) Start(ctx context.Context, appCtx *appCtx.AppContext) error {
	if err := s.buildModuleServers(ctx, appCtx); err != nil {
		return err
	}
	defer s.stopModuleServers(ctx)

	srv, err := server.New(s.cfg.HttpConfig.Port)
	if err != nil {
		return err
	}

	return srv.ServeHTTPHandler(ctx, s.Routes(ctx, appCtx))
}

func (s *Server) buildModuleServers(ctx context.Context, appContext *appCtx.AppContext) error {
	accountServer, err := accountassembly.BuildServer(appContext)
	if err != nil {
		return fmt.Errorf("build account server failed: %w", err)
	}

	roomServer, err := roomassembly.BuildServer(ctx, appContext)
	if err != nil {
		return fmt.Errorf("build room server failed: %w", err)
	}

	s.moduleServers = []moduleServer{accountServer, roomServer}
	return nil
}

func (s *Server) registerPublicAPI() {
	apiV1 := s.router.Group("/api/v1")
	for _, moduleServer := range s.moduleServers {
		moduleServer.RegisterPublicRoutes(apiV1)
	}
}

func (s *Server) registerPrivateAPI() {
	apiV1 := s.router.Group("/api/v1")
	apiV1.Use(middleware.AuthenMiddleware(s.appCtx))
	for _, moduleServer := range s.moduleServers {
		moduleServer.RegisterPrivateRoutes(apiV1)
	}
}

func (s *Server) stopModuleServers(ctx context.Context) {
	for i := len(s.moduleServers) - 1; i >= 0; i-- {
		if err := s.moduleServers[i].Stop(ctx); err != nil {
			logging.FromContext(ctx).Errorw("failed to stop http module server", "error", err)
		}
	}
}
