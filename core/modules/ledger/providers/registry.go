package providers

import (
	"fmt"
	"strings"
	"sync"
)

type ProviderRegistry struct {
	mu        sync.RWMutex
	providers map[string]PaymentProvider
}

func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[string]PaymentProvider),
	}
}

func (r *ProviderRegistry) Register(provider PaymentProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[strings.ToLower(strings.TrimSpace(provider.Name()))] = provider
}

func (r *ProviderRegistry) Get(name string) (PaymentProvider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, ok := r.providers[strings.ToLower(strings.TrimSpace(name))]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrProviderNotFound, name)
	}
	return provider, nil
}
