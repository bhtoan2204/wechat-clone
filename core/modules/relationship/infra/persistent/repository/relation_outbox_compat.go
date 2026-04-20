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

func persistRelationOutboxEvents(
	ctx context.Context,
	db *gorm.DB,
	serializer eventpkg.Serializer,
	events []eventpkg.Event,
) error {
	if len(events) == 0 {
		return nil
	}

	outboxModels := make([]models.RelationOutboxEvent, 0, len(events))
	for _, ev := range events {
		eventData, err := serializeRelationOutboxEventData(serializer, ev.EventData)
		if err != nil {
			return stackErr.Error(err)
		}

		outboxModels = append(outboxModels, models.RelationOutboxEvent{
			AggregateID:   ev.AggregateID,
			AggregateType: ev.AggregateType,
			Version:       int64(ev.Version),
			EventName:     ev.EventName,
			EventData:     eventData,
		})
	}

	if err := db.WithContext(ctx).Create(&outboxModels).Error; err != nil {
		return stackErr.Error(err)
	}
	return nil
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
