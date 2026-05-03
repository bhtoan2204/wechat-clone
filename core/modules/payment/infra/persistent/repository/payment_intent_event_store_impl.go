package repository

import (
	"context"
	"fmt"
	"time"
	"wechat-clone/core/modules/payment/infra/persistent/model"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type paymentOutboxEventStore struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func (s *paymentOutboxEventStore) Append(ctx context.Context, evt eventpkg.Event) error {
	if s == nil || s.db == nil {
		return stackErr.Error(eventpkg.ErrEventStoreNil)
	}

	serializer := s.serializer
	if serializer == nil {
		serializer = eventpkg.NewSerializer()
	}
	data, err := serializer.Marshal(evt.EventData)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal payment outbox event data failed: %w", err))
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return stackErr.Error(s.db.WithContext(ctx).Create(&model.PaymentOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}).Error)
}
