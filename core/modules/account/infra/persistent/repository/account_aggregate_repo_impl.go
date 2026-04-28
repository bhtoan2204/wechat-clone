package repos

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"wechat-clone/core/modules/account/domain/aggregate"
	"wechat-clone/core/modules/account/domain/entity"
	accountrepos "wechat-clone/core/modules/account/domain/repos"
	accountcache "wechat-clone/core/modules/account/infra/cache"
	"wechat-clone/core/modules/account/infra/persistent/models"
	sharedcache "wechat-clone/core/shared/infra/cache"
	shareddb "wechat-clone/core/shared/infra/db"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type accountAggregateRepoImpl struct {
	db              *gorm.DB
	serializer      eventpkg.Serializer
	projectionCache accountcache.AccountCache
	outboxPublisher eventpkg.Publisher
	afterCommit     afterCommitRegistrar
	ownsTransaction bool
}

type accountOutboxEventStore struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func NewAccountAggregateRepoImpl(
	db *gorm.DB,
	cache sharedcache.Cache,
	afterCommit afterCommitRegistrar,
	ownsTransaction bool,
) accountrepos.AccountAggregateRepository {
	if afterCommit == nil {
		afterCommit = func(ctx context.Context, fn func(context.Context)) {
			if fn != nil {
				fn(ctx)
			}
		}
	}

	serializer := newAccountAggregateSerializer()
	return &accountAggregateRepoImpl{
		db:              db,
		serializer:      serializer,
		projectionCache: accountcache.NewAccountCache(cache),
		outboxPublisher: eventpkg.NewPublisher(&accountOutboxEventStore{
			db:         db,
			serializer: serializer,
		}),
		afterCommit:     afterCommit,
		ownsTransaction: ownsTransaction,
	}
}

func (r *accountAggregateRepoImpl) Load(ctx context.Context, accountID string) (*aggregate.AccountAggregate, error) {
	return r.load(ctx, accountID, nil)
}

func (r *accountAggregateRepoImpl) LoadByEmail(ctx context.Context, email string) (*aggregate.AccountAggregate, error) {
	accountProjection, err := r.loadProjectionByEmail(ctx, email)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return r.load(ctx, accountProjection.ID, accountProjection)
}

func (r *accountAggregateRepoImpl) load(ctx context.Context, accountID string, accountProjection *entity.Account) (*aggregate.AccountAggregate, error) {
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

	var projectionErr error
	if accountProjection == nil {
		accountProjection, projectionErr = r.loadProjection(ctx, accountID)
	}
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

	save := func(tx *gorm.DB) error {
		if err := r.saveProjection(ctx, tx, snapshot); err != nil {
			return stackErr.Error(err)
		}

		previousPublisher := r.outboxPublisher
		r.outboxPublisher = eventpkg.NewPublisher(&accountOutboxEventStore{
			db:         tx,
			serializer: r.serializer,
		})
		defer func() {
			r.outboxPublisher = previousPublisher
		}()

		if err := r.publishOutboxEvents(ctx, agg); err != nil {
			return stackErr.Error(err)
		}
		return nil
	}

	if r.ownsTransaction {
		if err := r.db.WithContext(ctx).Transaction(save); err != nil {
			return stackErr.Error(err)
		}
	} else {
		if err := save(r.db.WithContext(ctx)); err != nil {
			return stackErr.Error(err)
		}
	}

	r.syncCacheAfterCommit(ctx, snapshot)
	return nil
}

func (r *accountAggregateRepoImpl) saveProjection(ctx context.Context, db *gorm.DB, snapshot *entity.Account) error {
	if snapshot == nil {
		return stackErr.Error(fmt.Errorf("account snapshot is nil"))
	}
	if err := db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"email",
				"password",
				"display_name",
				"username",
				"avatar_object_key",
				"status",
				"email_verified_at",
				"last_login_at",
				"password_changed_at",
				"banned_reason",
				"banned_until",
				"updated_at",
			}),
		}).
		Create(accountToProjectionModel(snapshot)).Error; err != nil {
		if shareddb.IsUniqueConstraintError(err) {
			return stackErr.Error(mapAccountUniqueConstraintError(err))
		}
		return stackErr.Error(fmt.Errorf("save account projection failed: %w", err))
	}
	return nil
}

func mapAccountUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	errMsg := strings.ToLower(err.Error())
	switch {
	case strings.Contains(errMsg, "username"):
		return accountrepos.ErrAccountUsernameAlreadyExists
	case strings.Contains(errMsg, "email"):
		return accountrepos.ErrAccountEmailAlreadyExists
	default:
		return accountrepos.ErrAccountEmailAlreadyExists
	}
}

func (r *accountAggregateRepoImpl) syncCacheAfterCommit(ctx context.Context, account *entity.Account) {
	if r == nil || r.afterCommit == nil || account == nil {
		return
	}
	r.afterCommit(ctx, func(hookCtx context.Context) {
		_ = r.projectionCache.Set(hookCtx, account)
		_ = r.projectionCache.SetByEmail(hookCtx, account)
	})
}

func (r *accountAggregateRepoImpl) loadProjection(ctx context.Context, accountID string) (*entity.Account, error) {
	var model models.AccountModel
	if err := r.db.WithContext(ctx).
		Where("id = ?", accountID).
		First(&model).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	return projectionModelToAccount(&model)
}

func (r *accountAggregateRepoImpl) loadProjectionByEmail(ctx context.Context, email string) (*entity.Account, error) {
	var model models.AccountModel
	if err := r.db.WithContext(ctx).
		Where("email = ?", email).
		First(&model).Error; err != nil {
		return nil, stackErr.Error(err)
	}

	return projectionModelToAccount(&model)
}

func (r *accountAggregateRepoImpl) publishOutboxEvents(ctx context.Context, agg *aggregate.AccountAggregate) error {
	if agg == nil || len(agg.Root().CloneEvents()) == 0 {
		return nil
	}
	if r == nil || r.outboxPublisher == nil {
		return stackErr.Error(eventpkg.ErrEventStoreNil)
	}
	return stackErr.Error(r.outboxPublisher.PublishAggregate(ctx, agg))
}

func (s *accountOutboxEventStore) Append(ctx context.Context, evt eventpkg.Event) error {
	if s == nil || s.db == nil {
		return stackErr.Error(eventpkg.ErrEventStoreNil)
	}

	serializer := s.serializer
	if serializer == nil {
		serializer = eventpkg.NewSerializer()
	}
	data, err := serializer.Marshal(evt.EventData)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal account event data failed: %w", err))
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return stackErr.Error(s.db.WithContext(ctx).Create(&models.AccountOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		CreatedAt:     createdAt,
	}).Error)
}

func (r *accountAggregateRepoImpl) toDomainEvent(eventModel models.AccountOutboxEventModel) (eventpkg.Event, error) {
	payloadFactory, ok := r.serializer.Type(eventModel.AggregateType, eventModel.EventName)
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
