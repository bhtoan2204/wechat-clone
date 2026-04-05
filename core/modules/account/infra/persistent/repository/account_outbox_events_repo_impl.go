package repos

import (
	"context"
	"fmt"
	"go-socket/core/modules/account/domain/repos"
	"go-socket/core/modules/account/infra/persistent/models"
	eventpkg "go-socket/core/shared/pkg/event"
	"time"

	"gorm.io/gorm"
)

type accountOutboxEventsRepoImpl struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func NewAccountOutboxEventsRepoImpl(db *gorm.DB) repos.AccountOutboxEventsRepository {
	return &accountOutboxEventsRepoImpl{
		db:         db,
		serializer: eventpkg.NewSerializer(),
	}
}

func (a *accountOutboxEventsRepoImpl) Append(ctx context.Context, evt eventpkg.Event) error {
	data, err := a.serializer.Marshal(evt.EventData)
	if err != nil {
		return fmt.Errorf("marshal event data failed: %v", err)
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return a.db.WithContext(ctx).Create(&models.AccountOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		CreatedAt:     createdAt,
	}).Error
}
