package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"go-socket/core/modules/payment/domain/aggregate"
	"go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/infra/persistent/model"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const paymentBalanceSnapshotInterval = 50

type paymentBalanceAggregateRepoImpl struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func NewPaymentBalanceAggregateRepoImpl(db *gorm.DB) repos.PaymentBalanceAggregateRepository {
	return &paymentBalanceAggregateRepoImpl{
		db:         db,
		serializer: newPaymentBalanceSerializer(),
	}
}

func (p *paymentBalanceAggregateRepoImpl) Load(ctx context.Context, accountID string) (*aggregate.PaymentBalanceAggregate, error) {
	if accountID == "" {
		return nil, fmt.Errorf("account id is empty")
	}

	agg, err := aggregate.NewPaymentBalanceAggregate(accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var aggregateModel model.PaymentAggregateModel
	err = p.db.WithContext(ctx).
		Where("aggregate_id = ?", accountID).
		First(&aggregateModel).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return agg, nil
		}
		return nil, stackErr.Error(err)
	}

	snapshotVersion, err := p.loadSnapshot(ctx, agg)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var eventModels []model.PaymentEventModel
	query := p.db.WithContext(ctx).
		Where("aggregate_id = ?", accountID)
	if snapshotVersion > 0 {
		query = query.Where("version > ?", snapshotVersion)
	}
	if err := query.
		Order("version ASC").
		Find(&eventModels).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	if len(eventModels) == 0 {
		if snapshotVersion == 0 {
			return nil, fmt.Errorf("payment aggregate has no snapshot and no events: account_id=%s version=%d", accountID, aggregateModel.Version)
		}
		if agg.Root().Version() != aggregateModel.Version {
			return nil, fmt.Errorf("payment aggregate version mismatch: aggregate=%d snapshot=%d", aggregateModel.Version, agg.Root().Version())
		}
		return agg, nil
	}

	events := make([]eventpkg.Event, 0, len(eventModels))
	for _, eventModel := range eventModels {
		domainEvent, err := p.toDomainEvent(eventModel)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		events = append(events, domainEvent)
	}

	if err := agg.LoadFromHistory(agg, events); err != nil {
		return nil, stackErr.Error(err)
	}
	if agg.Root().Version() != aggregateModel.Version {
		return nil, fmt.Errorf("payment aggregate version mismatch: aggregate=%d events=%d", aggregateModel.Version, agg.Root().Version())
	}

	return agg, nil
}

func (p *paymentBalanceAggregateRepoImpl) Save(ctx context.Context, agg *aggregate.PaymentBalanceAggregate) error {
	if agg == nil {
		return fmt.Errorf("payment aggregate is nil")
	}

	events := agg.Root().CloneEvents()
	if len(events) == 0 {
		return nil
	}

	if err := p.persistAggregateVersion(ctx, agg); err != nil {
		return stackErr.Error(err)
	}

	for _, evt := range events {
		eventModel, err := p.buildEventModel(evt)
		if err != nil {
			return stackErr.Error(err)
		}

		if err := p.db.WithContext(ctx).Create(&eventModel).Error; err != nil {
			return stackErr.Error(fmt.Errorf("create payment event failed: %v", err))
		}
	}

	if shouldCreatePaymentSnapshot(agg.Root().BaseVersion(), agg.Root().Version()) {
		if err := p.createSnapshot(ctx, agg); err != nil {
			return stackErr.Error(err)
		}
	}

	agg.Root().Update()
	return nil
}

func (p *paymentBalanceAggregateRepoImpl) persistAggregateVersion(ctx context.Context, agg *aggregate.PaymentBalanceAggregate) error {
	root := agg.Root()
	now := time.Now().UTC()

	if root.BaseVersion() == 0 {
		if err := p.db.WithContext(ctx).Create(&model.PaymentAggregateModel{
			ID:            root.AggregateID(),
			AggregateID:   root.AggregateID(),
			AggregateType: root.AggregateType(),
			Version:       root.Version(),
			CreatedAt:     now,
			UpdatedAt:     now,
		}).Error; err != nil {
			return stackErr.Error(fmt.Errorf("create payment aggregate failed: %v", err))
		}
		return nil
	}

	result := p.db.WithContext(ctx).
		Model(&model.PaymentAggregateModel{}).
		Where("aggregate_id = ? AND version = ?", root.AggregateID(), root.BaseVersion()).
		UpdateColumns(map[string]interface{}{
			"version":    root.Version(),
			"updated_at": now,
		})
	if result.Error != nil {
		return stackErr.Error(fmt.Errorf("update payment aggregate version failed: %v", result.Error))
	}
	if result.RowsAffected == 0 {
		return stackErr.Error(repos.ErrPaymentVersionConflict)
	}

	return nil
}

func (p *paymentBalanceAggregateRepoImpl) buildEventModel(evt eventpkg.Event) (model.PaymentEventModel, error) {
	data, err := p.serializer.Marshal(evt.EventData)
	if err != nil {
		return model.PaymentEventModel{}, fmt.Errorf("marshal event data failed: %v", err)
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	eventID := uuid.NewString()

	return model.PaymentEventModel{
		ID:            eventID,
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}, nil
}

func (p *paymentBalanceAggregateRepoImpl) loadSnapshot(ctx context.Context, agg *aggregate.PaymentBalanceAggregate) (int, error) {
	var snapshot model.PaymentBalanceSnapshotModel
	err := p.db.WithContext(ctx).
		Where("aggregate_id = ?", agg.Root().AggregateID()).
		Order("version DESC").
		First(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return 0, nil
		}
		return 0, fmt.Errorf("load payment snapshot failed: %v", err)
	}

	if err := p.restoreSnapshot(agg, snapshot); err != nil {
		return 0, err
	}

	return snapshot.Version, nil
}

func (p *paymentBalanceAggregateRepoImpl) restoreSnapshot(agg *aggregate.PaymentBalanceAggregate, snapshot model.PaymentBalanceSnapshotModel) error {
	if err := p.serializer.Unmarshal([]byte(snapshot.State), agg); err != nil {
		return fmt.Errorf("unmarshal payment snapshot failed: %v", err)
	}

	if agg.AccountID == "" {
		agg.AccountID = snapshot.AggregateID
	}
	agg.SetInternal(snapshot.AggregateID, snapshot.Version, snapshot.Version)
	return nil
}

func (p *paymentBalanceAggregateRepoImpl) createSnapshot(ctx context.Context, agg *aggregate.PaymentBalanceAggregate) error {
	state, err := p.serializer.Marshal(agg)
	if err != nil {
		return fmt.Errorf("marshal payment snapshot failed: %v", err)
	}

	snapshot := model.PaymentBalanceSnapshotModel{
		ID:          uuid.NewString(),
		AggregateID: agg.Root().AggregateID(),
		Version:     agg.Root().Version(),
		State:       string(state),
		CreatedAt:   time.Now().UTC(),
	}
	if err := p.db.WithContext(ctx).Create(&snapshot).Error; err != nil {
		return fmt.Errorf("create payment snapshot failed: %v", err)
	}

	return nil
}

func (p *paymentBalanceAggregateRepoImpl) toDomainEvent(eventModel model.PaymentEventModel) (eventpkg.Event, error) {
	payloadFactory, ok := p.serializer.Type(eventModel.AggregateType, eventModel.EventName)
	if !ok {
		return eventpkg.Event{}, fmt.Errorf("unsupported payment event: aggregate_type=%s event_name=%s", eventModel.AggregateType, eventModel.EventName)
	}
	payload := clonePaymentPayload(payloadFactory())
	if payload == nil {
		return eventpkg.Event{}, fmt.Errorf("payment event payload prototype is nil")
	}
	if err := p.serializer.Unmarshal([]byte(eventModel.EventData), payload); err != nil {
		return eventpkg.Event{}, err
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

func newPaymentBalanceSerializer() eventpkg.Serializer {
	serializer := eventpkg.NewSerializer()
	if err := serializer.RegisterAggregate(&aggregate.PaymentBalanceAggregate{}); err != nil {
		panic(fmt.Sprintf("register payment aggregate serializer failed: %v", err))
	}
	return serializer
}

func shouldCreatePaymentSnapshot(baseVersion, newVersion int) bool {
	if newVersion <= 0 {
		return false
	}
	if baseVersion == 0 {
		return true
	}
	return newVersion/paymentBalanceSnapshotInterval > baseVersion/paymentBalanceSnapshotInterval
}

func clonePaymentPayload(prototype interface{}) interface{} {
	prototypeType := reflect.TypeOf(prototype)
	if prototypeType == nil {
		return nil
	}
	if prototypeType.Kind() == reflect.Ptr {
		return reflect.New(prototypeType.Elem()).Interface()
	}
	return reflect.New(prototypeType).Interface()
}
