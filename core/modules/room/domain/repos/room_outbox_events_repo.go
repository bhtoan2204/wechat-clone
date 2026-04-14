package repos

import (
	"context"
	eventpkg "go-socket/core/shared/pkg/event"
)

//go:generate mockgen -package=repos -destination=room_outbox_events_repo_mock.go -source=room_outbox_events_repo.go
type RoomOutboxEventsRepository interface {
	Append(ctx context.Context, event eventpkg.Event) error
}
