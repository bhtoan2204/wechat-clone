package http

import (
	"context"
	"net/http"
	"net/http/httputil"
	"os"
	"strings"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/logging"
	baseserver "wechat-clone/core/shared/pkg/server"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/transport/http/middleware"

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
	r.Use(middleware.SetRequestID())
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

	if os.Getenv("ENVIRONMENT") != "production" {
		s.prepareSwaggerDocs(ctx)
		s.registerSwaggerRoutes()
	}

	s.registerStorageProxy()

	// public api
	s.registerPublicAPI()
	s.registerPrivateAPI()
	return r
}

func (s *Server) Start(ctx context.Context, appCtx *appCtx.AppContext) error {
	srv, err := baseserver.New(s.cfg.ServerConfig.Port)
	if err != nil {
		return stackErr.Error(err)
	}

	return s.StartWithServer(ctx, appCtx, srv)
}

func (s *Server) StartWithServer(ctx context.Context, appCtx *appCtx.AppContext, srv *baseserver.Server) error {
	if err := s.buildModuleServers(ctx, appCtx); err != nil {
		return stackErr.Error(err)
	}
	defer s.stopModuleServers(ctx)

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
		moduleServer.RegisterSocketRoutes(apiV1)
	}
}

func (s *Server) stopModuleServers(ctx context.Context) {
	for i := len(s.moduleServers) - 1; i >= 0; i-- {
		if err := s.moduleServers[i].Stop(ctx); err != nil {
			logging.FromContext(ctx).Errorw("failed to stop http module server", zap.Error(err))
		}
	}
}

func (s *Server) registerStorageProxy() {
	upstream := strings.TrimSpace(s.cfg.StorageConfig.MinIOEndpoint)
	bucket := strings.TrimSpace(s.cfg.StorageConfig.MinIOBucket)
	if upstream == "" || bucket == "" {
		return
	}

	prefix := "/storage/" + strings.TrimPrefix(bucket, "/")
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			if s.cfg.StorageConfig.MinIOUseSSL {
				req.URL.Scheme = "https"
			}
			req.URL.Host = upstream
			req.Host = upstream
			req.Header.Set("Host", upstream)
			req.URL.Path = strings.TrimPrefix(req.URL.Path, "/storage")
			if req.URL.Path == "" {
				req.URL.Path = "/"
			}
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, "Bad Gateway: MinIO is unavailable", http.StatusBadGateway)
		},
	}

	s.router.Any(prefix, gin.WrapH(proxy))
	s.router.Any(prefix+"/*path", gin.WrapH(proxy))
}
