package repository

import (
	"context"
	"fmt"
	"time"
	"wechat-clone/core/modules/room/infra/persistent/models"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type roomOutboxEventsRepoImpl struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func NewRoomOutboxEventsRepoImpl(db *gorm.DB) eventpkg.Store {
	return &roomOutboxEventsRepoImpl{
		db:         db,
		serializer: eventpkg.NewSerializer(),
	}
}

func (r *roomOutboxEventsRepoImpl) Append(ctx context.Context, evt eventpkg.Event) error {
	model, err := r.toModel(evt)
	if err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(r.db.WithContext(ctx).Create(model).Error)
}

func (r *roomOutboxEventsRepoImpl) toModel(evt eventpkg.Event) (*models.RoomOutboxEventModel, error) {
	data, err := r.serializer.Marshal(evt.EventData)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("marshal event data failed: %w", err))
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return &models.RoomOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}, nil
}
