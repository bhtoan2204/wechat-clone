package repos

import (
	"context"

	eventpkg "go-socket/core/shared/pkg/event"
)

//go:generate mockgen -package=repos -destination=ledger_outbox_events_repo_mock.go -source=ledger_outbox_events_repo.go
type LedgerOutboxEventsRepository interface {
	Append(ctx context.Context, event eventpkg.Event) error
}
