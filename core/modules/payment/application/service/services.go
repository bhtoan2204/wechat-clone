package service

import (
	"go-socket/core/modules/payment/providers"
)

type Services interface {
	ProviderService() ProviderService
}

type services struct {
	providerService ProviderService
}

func NewServices(providerRegistry *providers.ProviderRegistry) Services {
	providerSvc := newProviderService(providerRegistry)
	return &services{
		providerService: providerSvc,
	}
}

func (s *services) ProviderService() ProviderService {
	return s.providerService
}
