package webpush

import (
	"context"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"
	"net/http"

	lib "github.com/SherClockHolmes/webpush-go"
)

var sendNotification = func(
	ctx context.Context,
	payload []byte,
	subscription *lib.Subscription,
	options *lib.Options,
) (*http.Response, error) {
	return lib.SendNotificationWithContext(ctx, payload, subscription, options)
}

//go:generate mockgen -package=webpush -destination=webpush_mock.go -source=webpush.go
type WebPushService interface {
	Send(ctx context.Context, payload []byte, subscription *lib.Subscription) error
}

type webPushService struct {
	vapidPublicKey  string
	vapidPrivateKey string
	ttl             int
}

func NewWebPushService(cfg *config.Config) WebPushService {
	return &webPushService{
		vapidPublicKey:  cfg.WebPushConfig.VAPIDPublicKey,
		vapidPrivateKey: cfg.WebPushConfig.VAPIDPrivateKey,
		ttl:             cfg.WebPushConfig.TTL,
	}
}

func (s *webPushService) Send(
	ctx context.Context,
	payload []byte,
	subscription *lib.Subscription,
) error {
	resp, err := sendNotification(ctx, payload, subscription, &lib.Options{
		VAPIDPublicKey:  s.vapidPublicKey,
		VAPIDPrivateKey: s.vapidPrivateKey,
		TTL:             s.ttl,
	})
	if err != nil {
		return stackErr.Error(err)
	}

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	return nil
}
