package elasticsearch

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"

	es8 "github.com/elastic/go-elasticsearch/v8"
)

func NewClient(cfg config.ElasticsearchConfig) (*es8.Client, error) {
	if !cfg.Enabled {
		return nil, nil
	}

	addresses := splitAddresses(cfg.Addresses)
	if len(addresses) == 0 {
		addresses = []string{"http://localhost:9200"}
	}

	client, err := es8.NewClient(es8.Config{
		Addresses: addresses,
		Username:  strings.TrimSpace(cfg.Username),
		Password:  cfg.Password,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Duration(cfg.ResponseHeaderTimeoutSec) * time.Second,
			DialContext: (&net.Dialer{
				Timeout: time.Duration(cfg.ConnectTimeoutSeconds) * time.Second,
			}).DialContext,
		},
	})
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("create elasticsearch client failed: %v", err))
	}

	return client, nil
}

func splitAddresses(raw string) []string {
	parts := strings.Split(raw, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if item := strings.TrimSpace(part); item != "" {
			result = append(result, item)
		}
	}
	return result
}
