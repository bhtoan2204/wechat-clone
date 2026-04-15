package repository

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go-socket/core/modules/ledger/infra/persistent/model"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type dbTX interface {
	WithContext(ctx context.Context) *gorm.DB
}

type ledgerEventStore interface {
	CreateIfNotExist(ctx context.Context, aggregateID, aggregateType string) error
	CheckAndUpdateVersion(ctx context.Context, aggregateID, aggregateType string, baseVersion, newVersion int) (bool, error)
	Append(ctx context.Context, evt eventpkg.Event) error
	Get(ctx context.Context, aggregateID, aggregateType string, afterVersion int, agg eventpkg.Aggregate) error
	CreateSnapshot(ctx context.Context, agg eventpkg.Aggregate) error
	ReadSnapshot(ctx context.Context, aggregateID, aggregateType string, agg eventpkg.Aggregate) (bool, error)
}

type ledgerEventStoreImpl struct {
	db         dbTX
	serializer eventpkg.Serializer
}

func newLedgerEventStore(dbTX dbTX, serializer eventpkg.Serializer) ledgerEventStore {
	return &ledgerEventStoreImpl{
		db:         dbTX,
		serializer: serializer,
	}
}

func (s *ledgerEventStoreImpl) CreateIfNotExist(ctx context.Context, aggregateID, aggregateType string) error {
	now := time.Now().UTC()
	err := s.db.WithContext(ctx).Create(&model.LedgerAggregateModel{
		ID:            ledgerAggregateModelID(aggregateType, aggregateID),
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		Version:       0,
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error
	if err != nil && isOracleUniqueConstraintError(err) {
		return nil
	}
	return mapError(err)
}

func (s *ledgerEventStoreImpl) CheckAndUpdateVersion(
	ctx context.Context,
	aggregateID string,
	aggregateType string,
	baseVersion int,
	newVersion int,
) (bool, error) {
	result := s.db.WithContext(ctx).
		Model(&model.LedgerAggregateModel{}).
		Where("aggregate_id = ? AND aggregate_type = ? AND version = ?", aggregateID, aggregateType, baseVersion).
		Updates(map[string]interface{}{
			"version":    newVersion,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return false, mapError(result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (s *ledgerEventStoreImpl) Append(ctx context.Context, evt eventpkg.Event) error {
	data, err := s.serializer.Marshal(evt.EventData)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal ledger event data failed: %v", err))
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return mapError(s.db.WithContext(ctx).Create(&model.LedgerEventModel{
		ID:            uuid.NewString(),
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}).Error)
}

func (s *ledgerEventStoreImpl) Get(
	ctx context.Context,
	aggregateID string,
	aggregateType string,
	afterVersion int,
	agg eventpkg.Aggregate,
) error {
	var eventModels []model.LedgerEventModel
	query := s.db.WithContext(ctx).
		Where("aggregate_id = ? AND aggregate_type = ?", aggregateID, aggregateType).
		Order("version ASC")
	if afterVersion > 0 {
		query = query.Where("version > ?", afterVersion)
	}
	if err := query.Find(&eventModels).Error; err != nil {
		return stackErr.Error(mapError(err))
	}
	if len(eventModels) == 0 {
		if afterVersion == 0 {
			return stackErr.Error(ErrNotFound)
		}
		return nil
	}

	events := make([]eventpkg.Event, 0, len(eventModels))
	for _, eventModel := range eventModels {
		evt, err := s.toDomainEvent(eventModel)
		if err != nil {
			return stackErr.Error(err)
		}
		events = append(events, evt)
	}

	return stackErr.Error(agg.Root().LoadFromHistory(agg, events))
}

func (s *ledgerEventStoreImpl) CreateSnapshot(ctx context.Context, agg eventpkg.Aggregate) error {
	data, err := s.serializer.Marshal(agg)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal ledger snapshot failed: %v", err))
	}

	root := agg.Root()
	return stackErr.Error(mapError(s.db.WithContext(ctx).Create(&model.LedgerSnapshotModel{
		ID:            fmt.Sprintf("%s:%s:%d", root.AggregateType(), root.AggregateID(), root.Version()),
		AggregateID:   root.AggregateID(),
		AggregateType: root.AggregateType(),
		Version:       root.Version(),
		SnapshotData:  string(data),
		CreatedAt:     time.Now().UTC(),
	}).Error))
}

func (s *ledgerEventStoreImpl) ReadSnapshot(ctx context.Context, aggregateID, aggregateType string, agg eventpkg.Aggregate) (bool, error) {
	var snapshot model.LedgerSnapshotModel
	err := s.db.WithContext(ctx).
		Where("aggregate_id = ? AND aggregate_type = ?", aggregateID, aggregateType).
		Order("version DESC").
		First(&snapshot).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, stackErr.Error(mapError(err))
	}

	if err := s.serializer.Unmarshal([]byte(snapshot.SnapshotData), agg); err != nil {
		return false, stackErr.Error(err)
	}
	agg.Root().SetInternal(aggregateID, snapshot.Version, snapshot.Version)
	return true, nil
}

func (s *ledgerEventStoreImpl) toDomainEvent(eventModel model.LedgerEventModel) (eventpkg.Event, error) {
	payloadFactory, ok := s.serializer.Type(eventModel.AggregateType, eventModel.EventName)
	if !ok {
		return eventpkg.Event{}, stackErr.Error(fmt.Errorf(
			"unsupported ledger event: aggregate_type=%s event_name=%s",
			eventModel.AggregateType,
			eventModel.EventName,
		))
	}

	payload := cloneEventPayload(payloadFactory())
	if payload == nil {
		return eventpkg.Event{}, stackErr.Error(fmt.Errorf("ledger event payload prototype is nil"))
	}
	if err := s.serializer.Unmarshal([]byte(eventModel.EventData), payload); err != nil {
		return eventpkg.Event{}, stackErr.Error(err)
	}

	return eventpkg.Event{
		AggregateID:   eventModel.AggregateID,
		AggregateType: eventModel.AggregateType,
		Version:       eventModel.Version,
		EventName:     eventModel.EventName,
		EventData:     payload,
		CreatedAt:     eventModel.CreatedAt.Unix(),
	}, nil
}

func cloneEventPayload(prototype interface{}) interface{} {
	prototypeType := reflect.TypeOf(prototype)
	if prototypeType == nil {
		return nil
	}
	if prototypeType.Kind() == reflect.Ptr {
		return reflect.New(prototypeType.Elem()).Interface()
	}
	return reflect.New(prototypeType).Interface()
}
