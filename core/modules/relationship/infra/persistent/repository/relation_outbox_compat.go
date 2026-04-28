package repository

import (
	"context"
	"errors"
	"fmt"

	"wechat-clone/core/modules/relationship/infra/persistent/models"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

func loadRelationOutboxAggregateVersion(db *gorm.DB, aggregateID, aggregateType string) (int, error) {
	var latest models.RelationOutboxEvent
	err := db.
		Where("aggregate_id = ? AND aggregate_type = ?", aggregateID, aggregateType).
		Order("version DESC, id DESC").
		First(&latest).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, stackErr.Error(err)
	}

	return int(latest.Version), nil
}

func (s *relationOutboxEventStore) Append(ctx context.Context, ev eventpkg.Event) error {
	if s == nil || s.db == nil {
		return stackErr.Error(eventpkg.ErrEventStoreNil)
	}

	eventData, err := serializeRelationOutboxEventData(s.serializer, ev.EventData)
	if err != nil {
		return stackErr.Error(err)
	}

	return stackErr.Error(s.db.WithContext(ctx).Create(&models.RelationOutboxEvent{
		AggregateID:   ev.AggregateID,
		AggregateType: ev.AggregateType,
		Version:       int64(ev.Version),
		EventName:     ev.EventName,
		EventData:     eventData,
	}).Error)
}

func serializeRelationOutboxEventData(serializer eventpkg.Serializer, data interface{}) (string, error) {
	if data == nil {
		return "", stackErr.Error(fmt.Errorf("event data is nil"))
	}

	switch v := data.(type) {
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		b, err := serializer.Marshal(data)
		if err != nil {
			return "", stackErr.Error(err)
		}
		return string(b), nil
	}
}
