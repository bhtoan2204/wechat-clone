package http

import (
	"context"
	appCtx "go-socket/core/context"
	"go-socket/core/shared/config"
	"go-socket/core/shared/constant"
	"go-socket/core/shared/infra/idempotency"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/server"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/transport/http/middleware"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var _ App = (*Server)(nil)

//go:generate mockgen -package=http -destination=server_mock.go -source=server.go
type App interface {
	Routes(ctx context.Context, appCtx *appCtx.AppContext) *gin.Engine
	Start(ctx context.Context, appCtx *appCtx.AppContext) error
}

type Server struct {
	cfg            *config.Config
	router         *gin.Engine
	httpServer     *http.Server
	moduleServers  []HTTPServer
	moduleBuilders []ModuleBuilder
	appCtx         *appCtx.AppContext
	swaggerJSON    []byte
	swaggerPath    string
	swaggerErr     error
}

func NewServer(cfg *config.Config, opts ...Option) *Server {
	s := &Server{
		cfg: cfg,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
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
	if s.cfg.ServerConfig.Environment == "prod" {
		r.Use(middleware.IdempotencyMiddleware(idemManager))
		r.Use(middleware.RateLimitMiddleware(cache))
	}
	r.Use(gin.CustomRecovery(func(c *gin.Context, err interface{}) {
		c.JSON(http.StatusInternalServerError, gin.H{"errors": gin.H{"error": "something went wrong"}})
	}))
	r.Use(middleware.ErrorMiddleware())

	// cors
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true
	corsConfig.AllowHeaders = []string{
		"*",
		"Origin",
		"Content-Length",
		"Content-Type",
		"Authorization",
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
	r.GET("/successful", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"data":     "successful",
				"clientIP": ctx.ClientIP(),
			},
		})
	})
	r.GET("/failure", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"data":     "failure",
				"clientIP": ctx.ClientIP(),
			},
		})
	})
	r.GET("/cancel", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"data":     "cancel",
				"clientIP": ctx.ClientIP(),
			},
		})
	})
	s.router = r
	s.appCtx = appCtx

	s.prepareSwaggerDocs(ctx)
	s.registerSwaggerRoutes()

	// public api
	s.registerPublicAPI()
	s.registerPrivateAPI()
	return r
}

func (s *Server) Start(ctx context.Context, appCtx *appCtx.AppContext) error {
	if err := s.buildModuleServers(ctx, appCtx); err != nil {
		return stackErr.Error(err)
	}
	defer s.stopModuleServers(ctx)

	srv, err := server.New(s.cfg.ServerConfig.Port)
	if err != nil {
		return stackErr.Error(err)
	}

	return srv.ServeHTTPHandler(ctx, s.Routes(ctx, appCtx))
}

func (s *Server) buildModuleServers(ctx context.Context, appContext *appCtx.AppContext) error {
	if len(s.moduleBuilders) == 0 {
		s.moduleServers = []HTTPServer{}
		return nil
	}

	servers, err := BuildModuleServers(ctx, appContext, s.moduleBuilders...)
	if err != nil {
		return stackErr.Error(err)
	}
	s.moduleServers = servers
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
			logging.FromContext(ctx).Errorw("failed to stop http module server", zap.Error(err))
		}
	}
}
