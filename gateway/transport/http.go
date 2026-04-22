package transport

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"fmt"
	"gateway/config"
	"gateway/infra/proxy"
	redisPkg "gateway/infra/redis"
	"gateway/pkg/cache"
	"gateway/pkg/idempotency"
	"gateway/pkg/logging"
	stackErr "gateway/pkg/stackErr"
	"gateway/transport/middleware"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync/atomic"

	"github.com/hashicorp/consul/api"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type HTTPTransport struct {
	cfg          *config.Config
	server       *http.Server
	consulClient *api.Client
	redisClient  *redis.Client
	cacheClient  cache.Cache
	nextBackend  atomic.Uint64
}

func NewHTTPTransport(ctx context.Context, cfg *config.Config) (*HTTPTransport, error) {
	consulClient, err := proxy.NewConsulClient(ctx, cfg)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	redisClient, err := redisPkg.NewStandaloneRedisClient(cfg)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	cacheClient := cache.New(redisClient, cache.DEFAULT_CACHE_EXPIRATION_TIME)

	return &HTTPTransport{
		cfg:          cfg,
		consulClient: consulClient,
		redisClient:  redisClient,
		cacheClient:  cacheClient,
	}, nil
}

func normalizeListenAddr(value string) string {
	if _, _, err := net.SplitHostPort(value); err == nil {
		return value
	}
	if !strings.Contains(value, ":") {
		return ":" + value
	}
	return value
}

func (t *HTTPTransport) nextTargetHost(services []*api.ServiceEntry) (string, bool) {
	if len(services) == 0 {
		return "", false
	}

	start := t.nextBackend.Add(1) - 1
	for offset := range len(services) {
		entry := services[(int(start)+offset)%len(services)]
		if entry == nil || entry.Service == nil {
			continue
		}

		address := strings.TrimSpace(entry.Service.Address)
		if address == "" && entry.Node != nil {
			address = strings.TrimSpace(entry.Node.Address)
		}
		if address == "" || entry.Service.Port == 0 {
			continue
		}

		return fmt.Sprintf("%s:%d", address, entry.Service.Port), true
	}

	return "", false
}

func parseEd25519PublicKey(value string) (ed25519.PublicKey, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(strings.TrimSpace(value))
	if err != nil {
		return nil, fmt.Errorf("decode base64 public key: %w", err)
	}

	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid ed25519 public key length: got %d, want %d", len(keyBytes), ed25519.PublicKeySize)
	}

	return ed25519.PublicKey(keyBytes), nil
}

func chain(h http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}

func (t *HTTPTransport) Start() error {
	log := logging.DefaultLogger()
	addr := normalizeListenAddr(t.cfg.HTTP.Port)
	publicKey, err := parseEd25519PublicKey(t.cfg.AuthConfig.AccessPublicKey)
	if err != nil {
		return stackErr.Error(err)
	}

	idemStore := idempotency.NewRedisStore(t.cacheClient)
	idemManager := idempotency.NewManager(
		idemStore,
		idempotency.DEFAULT_IDEMPOTENCY_LOCK_TTL,
		idempotency.DEFAULT_IDEMPOTENCY_DONE_TTL,
	)

	// For now, we don't have mesh networking, so we need to proxy the request to the target service
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			services, _, err := t.consulClient.Health().Service("wechat-clone", "", true, nil)
			targetHost, ok := t.nextTargetHost(services)
			if err != nil || !ok {
				req.URL.Scheme = "http"
				req.URL.Host = "service-not-found"
				return
			}

			req.URL.Scheme = "http"
			req.URL.Host = targetHost
			req.Host = targetHost
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Errorw("ErrorHandler", zap.Error(err), zap.Any("request", r))
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error": "Bad Gateway: Service 'wechat-clone' is currently unavailable or not found in Consul"}`))
		},
	}

	handler := chain(
		proxy,
		middleware.AuthMiddleware(publicKey),
		middleware.RateLimitMiddleware(t.cacheClient),
		middleware.IdempotencyMiddleware(idemManager),
	)

	t.server = &http.Server{
		Addr:    addr,
		Handler: handler,
	}
	return t.server.ListenAndServe()
}

func (t *HTTPTransport) Stop() error {
	return t.server.Shutdown(context.Background())
}
