package lock

import (
	"context"
	"errors"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"math/rand"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

//go:generate mockgen -package=lock -destination=lock_mock.go -source=lock.go
type Lock interface {
	AcquireLock(ctx context.Context, key, value string, expiration, retryDelay, timeout time.Duration) (bool, error)
	ReleaseLock(ctx context.Context, key, value string) (bool, error)
}

type lock struct {
	client *redis.Client
}

func NewLock(client *redis.Client) Lock {
	return &lock{client: client}
}

var releaseLockScript = redis.NewScript(`
	if redis.call("get", KEYS[1]) == ARGV[1] then
		redis.call("unlink", KEYS[1])
		return 1
	else
		return 0
	end
`)

func (l *lock) AcquireLock(ctx context.Context, key, value string, expiration, retryDelay, timeout time.Duration) (bool, error) {
	log := logging.FromContext(ctx)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		ok, err := l.client.SetNX(ctx, key, value, expiration).Result()
		if err != nil {
			log.Errorw("AcquireLock redis error", zap.String("key", key), zap.Error(err))
			return false, stackErr.Error(err)
		}
		if ok {
			return true, nil
		}

		jitter := time.Duration(rand.Int63n(int64(retryDelay / 2)))
		select {
		case <-time.After(retryDelay + jitter):
			// retry
		case <-ctx.Done():
			log.Warnw("AcquireLock timeout", zap.String("key", key))
			return false, stackErr.Error(errors.New("timeout acquiring lock"))
		}
	}
}

func (l *lock) ReleaseLock(ctx context.Context, key, value string) (bool, error) {
	log := logging.FromContext(ctx)

	res, err := releaseLockScript.Run(ctx, l.client, []string{key}, value).Result()
	if err != nil {
		log.Errorw("ReleaseLock lua error", zap.String("key", key), zap.Error(err))
		return false, stackErr.Error(err)
	}

	hit, ok := res.(int64)
	if !ok || hit == 0 {
		log.Warnw("ReleaseLock not hit", zap.String("key", key))
		return false, nil
	}
	return true, nil
}
