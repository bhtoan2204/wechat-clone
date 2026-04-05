package discovery

import (
	"context"
	"fmt"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/hashicorp/consul/api"
	"go.uber.org/zap"
)

func (c *consulClientImpl) RegisterService(ctx context.Context, serviceID string, serviceName string, serviceAddress string, servicePort int) error {
	log := logging.FromContext(ctx)
	healthCheckURL := fmt.Sprintf("http://%s:%d/health-check", serviceAddress, servicePort)

	registration := &api.AgentServiceRegistration{
		ID:      serviceID,
		Name:    serviceName,
		Address: serviceAddress,
		Port:    servicePort,
		Tags:    []string{"api", serviceName},
		Check: &api.AgentServiceCheck{
			HTTP:                           healthCheckURL,
			Method:                         "GET",
			Interval:                       "10s",
			Timeout:                        "5s",
			Notes:                          "Basic health check for " + serviceName,
			DeregisterCriticalServiceAfter: "1m",
		},
	}

	err := c.client.Agent().ServiceRegister(registration)
	if err != nil {
		log.Errorw("Failed to register service", "serviceID", serviceID, zap.Error(err))
		return stackErr.Error(err)
	}

	log.Infow("Service registered successfully", "serviceID", serviceID)
	return nil
}

func (c *consulClientImpl) UnregisterService(ctx context.Context, serviceID string) error {
	log := logging.FromContext(ctx)

	err := c.client.Agent().ServiceDeregister(serviceID)
	if err != nil {
		log.Errorw("Failed to unregister service", "serviceID", serviceID, zap.Error(err))
		return stackErr.Error(err)
	}

	return nil
}

func (c *consulClientImpl) GetService(ctx context.Context, serviceID string) (*api.AgentService, error) {
	services, err := c.client.Agent().Services()
	if err != nil {
		return nil, stackErr.Error(err)
	}

	service, exists := services[serviceID]
	if !exists {
		return nil, fmt.Errorf("service with ID %s not found on agent", serviceID)
	}

	return service, nil
}

func (c *consulClientImpl) GetServices(ctx context.Context) ([]*api.AgentService, error) {
	servicesMap, err := c.client.Agent().Services()
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var servicesList []*api.AgentService
	for _, svc := range servicesMap {
		servicesList = append(servicesList, svc)
	}

	return servicesList, nil
}

func (c *consulClientImpl) GetServiceHealth(ctx context.Context, serviceID string) ([]*api.HealthCheck, error) {
	log := logging.FromContext(ctx)
	svc, err := c.GetService(ctx, serviceID)
	if err != nil {
		log.Errorw("Failed to get service", "serviceID", serviceID, zap.Error(err))
		return nil, stackErr.Error(err)
	}

	checks, _, err := c.client.Health().Checks(svc.Service, &api.QueryOptions{
		Filter: fmt.Sprintf("ServiceID == `%s`", serviceID),
	})
	if err != nil {
		log.Errorw("Failed to get service health", "serviceID", serviceID, zap.Error(err))
		return nil, stackErr.Error(err)
	}

	return checks, nil
}
