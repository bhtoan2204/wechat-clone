package repository

import (
	"context"
	"fmt"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/infra/persistent/models"
	eventpkg "go-socket/core/shared/pkg/event"
	"time"

	"gorm.io/gorm"
)

type roomOutboxEventsRepoImpl struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func NewRoomOutboxEventsRepoImpl(db *gorm.DB) repos.RoomOutboxEventsRepository {
	return &roomOutboxEventsRepoImpl{
		db:         db,
		serializer: eventpkg.NewSerializer(),
	}
}

func (r *roomOutboxEventsRepoImpl) Append(ctx context.Context, evt eventpkg.Event) error {
	data, err := r.serializer.Marshal(evt.EventData)
	if err != nil {
		return fmt.Errorf("marshal event data failed: %v", err)
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return r.db.WithContext(ctx).Create(&models.RoomOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}).Error
}
