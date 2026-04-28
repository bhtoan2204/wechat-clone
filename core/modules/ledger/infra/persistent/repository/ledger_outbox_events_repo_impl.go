package repository

import (
	"context"
	"fmt"
	"time"

	"wechat-clone/core/modules/ledger/infra/persistent/model"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"
)

type ledgerOutboxEventsRepoImpl struct {
	db         dbTX
	serializer eventpkg.Serializer
}

func NewLedgerOutboxEventsRepoImpl(dbTX dbTX) eventpkg.Store {
	return &ledgerOutboxEventsRepoImpl{
		db:         dbTX,
		serializer: eventpkg.NewSerializer(),
	}
}

func (r *ledgerOutboxEventsRepoImpl) Append(ctx context.Context, evt eventpkg.Event) error {
	data, err := r.serializer.Marshal(evt.EventData)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal ledger outbox event failed: %w", err))
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
