package idempotency

import (
	"context"
	"time"
)

//go:generate mockgen -package=idempotency -destination=manager_mock.go -source=manager.go
type Store interface {
	TryLock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	MarkDone(ctx context.Context, key string, ttl time.Duration) error
	Release(ctx context.Context, key string) error
}

type Manager struct {
	store   Store
	lockTTL time.Duration
	doneTTL time.Duration
}

func NewManager(store Store, lockTTL, doneTTL time.Duration) *Manager {
	return &Manager{
		store:   store,
		lockTTL: lockTTL,
		doneTTL: doneTTL,
	}
}

func (m *Manager) Begin(ctx context.Context, key string) (bool, error) {
	if m == nil || m.store == nil {
		return true, nil
	}
	return m.store.TryLock(ctx, key, m.lockTTL)
}

func (m *Manager) End(ctx context.Context, key string, success bool) error {
	if m == nil || m.store == nil {
		return nil
	}
	if success {
		return m.store.MarkDone(ctx, key, m.doneTTL)
	}
	return m.store.Release(ctx, key)
}
