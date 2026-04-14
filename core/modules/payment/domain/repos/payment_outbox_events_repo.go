package repos

import (
	"context"
	eventpkg "go-socket/core/shared/pkg/event"
)

//go:generate mockgen -package=repos -destination=payment_outbox_events_repo_mock.go -source=payment_outbox_events_repo.go
type PaymentOutboxEventsRepository interface {
	Append(ctx context.Context, event eventpkg.Event) error
}
