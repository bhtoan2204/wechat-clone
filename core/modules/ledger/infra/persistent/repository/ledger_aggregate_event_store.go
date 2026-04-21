package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	"wechat-clone/core/modules/ledger/domain/entity"
	"wechat-clone/core/modules/ledger/domain/eventstore"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
	"wechat-clone/core/modules/ledger/infra/persistent/model"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type dbTX interface {
	WithContext(ctx context.Context) *gorm.DB
}

type ledgerEventStoreImpl struct {
	db         dbTX
	serializer eventpkg.Serializer
}

func newLedgerEventStore(dbTX dbTX, serializer eventpkg.Serializer) eventstore.LedgerEventStore {
	return &ledgerEventStoreImpl{
		db:         dbTX,
		serializer: serializer,
	}
}

func (s *ledgerEventStoreImpl) CreateIfNotExist(ctx context.Context, aggregateID, aggregateType string) error {
	now := time.Now().UTC()

	err := s.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "aggregate_id"},
				{Name: "aggregate_type"},
			},
			DoNothing: true,
		}).
		Create(&model.LedgerAggregateModel{
			ID:            uuid.NewString(),
			AggregateID:   aggregateID,
			AggregateType: aggregateType,
			Version:       0,
			CreatedAt:     now,
			UpdatedAt:     now,
		}).Error

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
			"version": newVersion,
		})
	if result.Error != nil {
		return false, mapError(result.Error)
	}
	return result.RowsAffected == 1, nil
}

func (s *ledgerEventStoreImpl) findPostedTransaction(
	ctx context.Context,
	aggregateID string,
	aggregateType string,
	transactionID string,
) (*entity.LedgerAccountPosting, error) {
	var postingModel model.LedgerPostedTransactionModel
	result := s.db.WithContext(ctx).
		Where("aggregate_id = ? AND aggregate_type = ? AND transaction_id = ?", aggregateID, aggregateType, transactionID).
		Limit(1).
		Find(&postingModel)
	if result.Error != nil {
		return nil, stackErr.Error(mapError(result.Error))
	}
	if result.RowsAffected == 0 {
		return nil, nil
	}

	return &entity.LedgerAccountPosting{
		TransactionID:         postingModel.TransactionID,
		ReferenceType:         postingModel.ReferenceType,
		ReferenceID:           postingModel.ReferenceID,
		CounterpartyAccountID: postingModel.CounterpartyAccountID,
		Currency:              postingModel.Currency,
		AmountDelta:           postingModel.AmountDelta,
		BookedAt:              postingModel.BookedAt.UTC(),
	}, nil
}

func (s *ledgerEventStoreImpl) ReservePostedTransaction(ctx context.Context, evt eventpkg.Event) error {
	posting, ok, err := ledgeraggregate.NewLedgerAccountPostingFromEvent(evt.AggregateID, evt.EventData)
	if err != nil {
		return stackErr.Error(err)
	}
	if !ok {
		return nil
	}

	if err := mapError(s.db.WithContext(ctx).Create(&model.LedgerPostedTransactionModel{
		ID:                    uuid.NewString(),
		AggregateID:           evt.AggregateID,
		AggregateType:         evt.AggregateType,
		TransactionID:         posting.TransactionID,
		ReferenceType:         posting.ReferenceType,
		ReferenceID:           posting.ReferenceID,
		CounterpartyAccountID: posting.CounterpartyAccountID,
		Currency:              posting.Currency,
		AmountDelta:           posting.AmountDelta,
		BookedAt:              posting.BookedAt.UTC(),
		CreatedAt:             time.Now().UTC(),
	}).Error); err == nil {
		return nil
	}
	if !errors.Is(err, ErrDuplicate) {
		return stackErr.Error(err)
	}

	existing, loadErr := s.findPostedTransaction(ctx, evt.AggregateID, evt.AggregateType, posting.TransactionID)
	if loadErr != nil {
		return stackErr.Error(loadErr)
	}
	if existing != nil && ledgeraggregate.SameLedgerAccountPosting(*existing, posting) {
		return stackErr.Error(ledgerrepos.ErrAlreadyApplied)
	}

	return stackErr.Error(fmt.Errorf(
		"existing ledger posting mismatch aggregate_id=%s aggregate_type=%s transaction_id=%s",
		evt.AggregateID,
		evt.AggregateType,
		posting.TransactionID,
	))
}

func (s *ledgerEventStoreImpl) Append(ctx context.Context, evt eventpkg.Event) error {
	data, err := s.serializer.Marshal(evt.EventData)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal ledger event data failed: %w", err))
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
		return stackErr.Error(fmt.Errorf("marshal ledger snapshot failed: %w", err))
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
	result := s.db.WithContext(ctx).
		Where("aggregate_id = ? AND aggregate_type = ?", aggregateID, aggregateType).
		Order("version DESC").
		Limit(1).
		Find(&snapshot)
	if result.Error != nil {
		return false, stackErr.Error(mapError(result.Error))
	}
	if result.RowsAffected == 0 {
		return false, nil
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
