package config

import (
	"context"
	"fmt"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/sethvargo/go-envconfig"
)

func LoadConfig(ctx context.Context) (*Config, error) {
	cfg := &Config{}
	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target:   cfg,
		Lookuper: envconfig.OsLookuper(),
	}); err != nil {
		return nil, stackErr.Error(fmt.Errorf("envconfig.ProcessWith has err=%v", err))
	}
	return cfg, nil
}
