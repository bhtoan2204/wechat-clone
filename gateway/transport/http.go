package transport

import (
	"context"
	"fmt"
	"gateway/config"
	"gateway/infra/proxy"
	stackErr "gateway/pkg/stackErr"
	"net"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync/atomic"

	"github.com/hashicorp/consul/api"
)

type HTTPTransport struct {
	cfg          *config.Config
	server       *http.Server
	consulClient *api.Client
	nextBackend  atomic.Uint64
}

func NewHTTPTransport(ctx context.Context, cfg *config.Config) (*HTTPTransport, error) {
	consulClient, err := proxy.NewConsulClient(ctx, cfg)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &HTTPTransport{
		cfg:          cfg,
		consulClient: consulClient,
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

func (t *HTTPTransport) Start() error {
	addr := normalizeListenAddr(t.cfg.HTTP.Port)

	// For now, we don't have mesh networking, so we need to proxy the request to the target service
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			services, _, err := t.consulClient.Health().Service("wechat-clone", "", true, nil)
			targetHost, ok := t.nextTargetHost(services)
			if err != nil || !ok {
				// Cố tình trỏ đến một host lỗi để ErrorHandler phía dưới bắt được
				req.URL.Scheme = "http"
				req.URL.Host = "service-not-found"
				return
			}

			req.URL.Scheme = "http"
			req.URL.Host = targetHost
			req.Host = targetHost
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			w.WriteHeader(http.StatusBadGateway)
			w.Write([]byte(`{"error": "Bad Gateway: Service 'wechat-clone' is currently unavailable or not found in Consul"}`))
		},
	}
	t.server = &http.Server{
		Addr:    addr,
		Handler: proxy,
	}
	return t.server.ListenAndServe()
}

func (t *HTTPTransport) Stop() error {
	return t.server.Shutdown(context.Background())
}
