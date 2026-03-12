package repos

import (
	"context"
	eventpkg "go-socket/core/shared/pkg/event"
)

type PaymentOutboxEventsRepository interface {
	Append(ctx context.Context, event eventpkg.Event) error
}
