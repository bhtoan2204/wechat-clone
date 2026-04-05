package discovery

import (
	"context"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

type ConsulClient interface {
	RegisterService(ctx context.Context, serviceID string, serviceName string, serviceAddress string, servicePort int) error
	UnregisterService(ctx context.Context, serviceID string) error
	GetService(ctx context.Context, serviceID string) (*api.AgentService, error)
	GetServices(ctx context.Context) ([]*api.AgentService, error)
	GetServiceHealth(ctx context.Context, serviceID string) ([]*api.HealthCheck, error)
}

type consulClientImpl struct {
	client *api.Client
}

func NewConsulClient(ctx context.Context, cfg *config.Config) (ConsulClient, error) {
	log := logging.FromContext(ctx)
	consulConfig := api.DefaultConfig()

	consulConfig.Address = cfg.ConsulConfig.Address
	consulConfig.Scheme = cfg.ConsulConfig.Scheme
	consulConfig.Token = cfg.ConsulConfig.Token
	consulConfig.Datacenter = cfg.ConsulConfig.DataCenter

	consulAPIClient, err := api.NewClient(consulConfig)
	if err != nil {
		log.Errorw("Failed to connect to Consul", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	return &consulClientImpl{
		client: consulAPIClient,
	}, nil
}
