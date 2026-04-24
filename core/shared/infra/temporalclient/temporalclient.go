package temporalclient

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.temporal.io/sdk/client"
)

const (
	temporalDialAttempts = 10
	temporalDialDelay    = 2 * time.Second
)

func NewTemporalClient(ctx context.Context, cfg *config.Config) (client.Client, error) {
	hostPort := resolveTemporalAddress(cfg.TemporalConfig)
	namespace := resolveTemporalNamespace(cfg.TemporalConfig)
	var lastErr error

	for attempt := 1; attempt <= temporalDialAttempts; attempt++ {
		c, err := client.Dial(client.Options{
			HostPort:  hostPort,
			Namespace: namespace,
		})
		if err == nil {
			return c, nil
		}
		lastErr = err

		if attempt == temporalDialAttempts {
			break
		}

		select {
		case <-ctx.Done():
			return nil, stackErr.Error(fmt.Errorf("dial temporal client canceled: %w", ctx.Err()))
		case <-time.After(temporalDialDelay):
		}
	}

	return nil, stackErr.Error(fmt.Errorf(
		"dial temporal client failed after %d attempts: %w",
		temporalDialAttempts,
		lastErr,
	))
}

func resolveTemporalAddress(cfg config.TemporalConfig) string {
	if address := strings.TrimSpace(cfg.Address); address != "" {
		return address
	}

	host := strings.TrimSpace(cfg.Host)
	if host == "" {
		host = "localhost"
	}

	port := cfg.Port
	if port <= 0 {
		port = 7233
	}

	return net.JoinHostPort(host, strconv.Itoa(port))
}

func resolveTemporalNamespace(cfg config.TemporalConfig) string {
	if namespace := strings.TrimSpace(cfg.Namespace); namespace != "" {
		return namespace
	}
	return "default"
}
