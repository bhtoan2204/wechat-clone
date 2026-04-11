package repos

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"time"

	"go-socket/core/modules/account/domain/aggregate"
	"go-socket/core/modules/account/domain/entity"
	accountrepos "go-socket/core/modules/account/domain/repos"
	accountcache "go-socket/core/modules/account/infra/cache"
	"go-socket/core/modules/account/infra/persistent/models"
	sharedcache "go-socket/core/shared/infra/cache"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

const (
	accountAggregateType       = "AccountAggregate"
	legacyAccountAggregateType = "account"
)

type accountAggregateRepoImpl struct {
	db               *gorm.DB
	serializer       eventpkg.Serializer
	projectionWriter *accountRepoImpl
}

func NewAccountAggregateRepoImpl(
	db *gorm.DB,
	cache sharedcache.Cache,
	afterCommit afterCommitRegistrar,
) accountrepos.AccountAggregateRepository {
	if afterCommit == nil {
		afterCommit = func(ctx context.Context, fn func(context.Context)) {
			if fn != nil {
				fn(ctx)
			}
		}
	}

	return &accountAggregateRepoImpl{
		db:         db,
		serializer: newAccountAggregateSerializer(),
		projectionWriter: &accountRepoImpl{
			db:           db,
			accountCache: accountcache.NewAccountCache(cache),
			afterCommit:  afterCommit,
		},
	}
}

func (r *accountAggregateRepoImpl) Load(ctx context.Context, accountID string) (*aggregate.AccountAggregate, error) {
	agg, err := aggregate.NewAccountAggregate(accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var eventModels []models.AccountOutboxEventModel
	if err := r.db.WithContext(ctx).
		Where("aggregate_id = ?", accountID).
		Order("version ASC, id ASC").
		Find(&eventModels).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	if len(eventModels) > 0 {
		events := make([]eventpkg.Event, 0, len(eventModels))
		for _, eventModel := range eventModels {
			domainEvent, err := r.toDomainEvent(eventModel)
			if err != nil {
				return nil, stackErr.Error(err)
			}
			events = append(events, domainEvent)
		}
		if err := agg.LoadFromHistory(agg, events); err != nil {
			return nil, stackErr.Error(err)
		}
	}

	accountProjection, projectionErr := r.loadProjection(ctx, accountID)
	if projectionErr == nil {
		if len(eventModels) == 0 {
			if err := agg.RestoreFromProjection(accountProjection, 0); err != nil {
				return nil, stackErr.Error(err)
			}
		} else {
			agg.MergeProjection(accountProjection)
		}
		return agg, nil
	}
	if errors.Is(projectionErr, gorm.ErrRecordNotFound) {
		if len(eventModels) == 0 {
			return nil, stackErr.Error(gorm.ErrRecordNotFound)
		}
		return agg, nil
	}
	return nil, stackErr.Error(projectionErr)
}

func (r *accountAggregateRepoImpl) Save(ctx context.Context, agg *aggregate.AccountAggregate) error {
	if agg == nil {
		return stackErr.Error(fmt.Errorf("account aggregate is nil"))
	}

	snapshot, err := agg.Snapshot()
	if err != nil {
		return stackErr.Error(err)
	}

	if err := r.db.WithContext(ctx).Save(r.projectionWriter.toModel(snapshot)).Error; err != nil {
		return stackErr.Error(fmt.Errorf("save account projection failed: %v", err))
	}
	r.projectionWriter.syncCacheAfterCommit(ctx, snapshot)

	events := agg.Root().CloneEvents()
	if len(events) == 0 {
		return nil
	}

	for _, evt := range events {
		eventModel, err := r.buildEventModel(evt)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := r.db.WithContext(ctx).Create(&eventModel).Error; err != nil {
			return stackErr.Error(fmt.Errorf("create account event failed: %v", err))
		}
	}

	agg.Root().Update()
	return nil
}

func (r *accountAggregateRepoImpl) loadProjection(ctx context.Context, accountID string) (*entity.Account, error) {
	var model models.AccountModel
	if err := r.db.WithContext(ctx).
		Where("id = ?", accountID).
		First(&model).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	mapper := &accountRepoImpl{}
	return mapper.toEntity(&model)
}

func (r *accountAggregateRepoImpl) buildEventModel(evt eventpkg.Event) (models.AccountOutboxEventModel, error) {
	data, err := r.serializer.Marshal(evt.EventData)
	if err != nil {
		return models.AccountOutboxEventModel{}, stackErr.Error(fmt.Errorf("marshal account event data failed: %v", err))
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return models.AccountOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		CreatedAt:     createdAt,
	}, nil
}

func (r *accountAggregateRepoImpl) toDomainEvent(eventModel models.AccountOutboxEventModel) (eventpkg.Event, error) {
	payloadFactory, ok := r.serializer.Type(eventModel.AggregateType, eventModel.EventName)
	if !ok && eventModel.AggregateType == legacyAccountAggregateType {
		payloadFactory, ok = r.serializer.Type(accountAggregateType, eventModel.EventName)
	}
	if !ok {
		return eventpkg.Event{}, stackErr.Error(fmt.Errorf(
			"unsupported account event: aggregate_type=%s event_name=%s",
			eventModel.AggregateType,
			eventModel.EventName,
		))
	}

	payload := cloneAccountPayload(payloadFactory())
	if payload == nil {
		return eventpkg.Event{}, stackErr.Error(fmt.Errorf("account event payload prototype is nil"))
	}
	if err := r.serializer.Unmarshal([]byte(eventModel.EventData), payload); err != nil {
		return eventpkg.Event{}, stackErr.Error(err)
	}

	aggregateType := eventModel.AggregateType
	if aggregateType == legacyAccountAggregateType {
		aggregateType = accountAggregateType
	}

	return eventpkg.Event{
		AggregateID:   eventModel.AggregateID,
		AggregateType: aggregateType,
		Version:       eventModel.Version,
		EventName:     eventModel.EventName,
		EventData:     payload,
		CreatedAt:     eventModel.CreatedAt.Unix(),
	}, nil
}

func newAccountAggregateSerializer() eventpkg.Serializer {
	serializer := eventpkg.NewSerializer()
	if err := serializer.RegisterAggregate(&aggregate.AccountAggregate{}); err != nil {
		panic(fmt.Sprintf("register account aggregate serializer failed: %v", err))
	}
	return serializer
}

func cloneAccountPayload(prototype interface{}) interface{} {
	prototypeType := reflect.TypeOf(prototype)
	if prototypeType == nil {
		return nil
	}
	if prototypeType.Kind() == reflect.Ptr {
		return reflect.New(prototypeType.Elem()).Interface()
	}
	return reflect.New(prototypeType).Interface()
}
