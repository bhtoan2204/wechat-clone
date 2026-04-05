package repository

import (
	"context"
	"fmt"
	"time"

	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/infra/persistent/model"
	eventpkg "go-socket/core/shared/pkg/event"

	"gorm.io/gorm"
)

type paymentOutboxEventsRepoImpl struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func NewPaymentOutboxEventsRepoImpl(db *gorm.DB) paymentrepos.PaymentOutboxEventsRepository {
	return &paymentOutboxEventsRepoImpl{
		db:         db,
		serializer: eventpkg.NewSerializer(),
	}
}

func (p *paymentOutboxEventsRepoImpl) Append(ctx context.Context, evt eventpkg.Event) error {
	data, err := p.serializer.Marshal(evt.EventData)
	if err != nil {
		return fmt.Errorf("marshal event data failed: %v", err)
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return p.db.WithContext(ctx).Create(&model.PaymentOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}).Error
}
