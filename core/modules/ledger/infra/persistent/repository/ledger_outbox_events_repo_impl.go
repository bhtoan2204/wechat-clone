package repository

import (
	"context"
	"fmt"
	"time"

	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	"go-socket/core/modules/ledger/infra/persistent/model"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

type ledgerOutboxEventsRepoImpl struct {
	db         dbTX
	serializer eventpkg.Serializer
}

func NewLedgerOutboxEventsRepoImpl(dbTX dbTX) ledgerrepos.LedgerOutboxEventsRepository {
	return &ledgerOutboxEventsRepoImpl{
		db:         dbTX,
		serializer: eventpkg.NewSerializer(),
	}
}

func (r *ledgerOutboxEventsRepoImpl) Append(ctx context.Context, evt eventpkg.Event) error {
	data, err := r.serializer.Marshal(evt.EventData)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal ledger outbox event failed: %v", err))
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return stackErr.Error(mapError(r.db.WithContext(ctx).Create(&model.LedgerOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}).Error))
}
