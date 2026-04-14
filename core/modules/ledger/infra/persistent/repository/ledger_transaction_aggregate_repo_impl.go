package repository

import (
	"context"
	"fmt"
	"time"

	ledgeraggregate "go-socket/core/modules/ledger/domain/aggregate"
	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	"go-socket/core/modules/ledger/infra/persistent/model"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ledgerTransactionAggregateRepoImpl struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func NewLedgerTransactionAggregateRepoImpl(db *gorm.DB) ledgerrepos.LedgerTransactionAggregateRepository {
	return &ledgerTransactionAggregateRepoImpl{
		db:         db,
		serializer: newLedgerTransactionSerializer(),
	}
}

func (r *ledgerTransactionAggregateRepoImpl) Save(ctx context.Context, aggregate *ledgeraggregate.LedgerTransactionAggregate) error {
	if aggregate == nil {
		return stackErr.Error(fmt.Errorf("ledger transaction aggregate is nil"))
	}

	root := aggregate.Root()
	events := root.CloneEvents()
	if len(events) == 0 {
		return nil
	}
	if root.BaseVersion() != 0 {
		return stackErr.Error(fmt.Errorf("ledger transaction aggregate does not support updates"))
	}

	transaction, err := aggregate.Snapshot()
	if err != nil {
		return stackErr.Error(err)
	}

	if err := r.persistAggregate(ctx, root); err != nil {
		return stackErr.Error(err)
	}

	entryIDs, err := r.persistProjection(ctx, transaction)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := r.persistEvents(ctx, events); err != nil {
		return stackErr.Error(err)
	}
	if err := aggregate.AssignEntryIDs(entryIDs); err != nil {
		return stackErr.Error(err)
	}

	root.Update()
	return nil
}

func (r *ledgerTransactionAggregateRepoImpl) persistAggregate(ctx context.Context, root *eventpkg.AggregateRoot) error {
	now := time.Now().UTC()
	return mapError(r.db.WithContext(ctx).Create(&model.LedgerAggregateModel{
		ID:            root.AggregateID(),
		AggregateID:   root.AggregateID(),
		AggregateType: root.AggregateType(),
		Version:       root.Version(),
		CreatedAt:     now,
		UpdatedAt:     now,
	}).Error)
}

func (r *ledgerTransactionAggregateRepoImpl) persistProjection(
	ctx context.Context,
	transaction *entity.LedgerTransaction,
) ([]int64, error) {
	if err := r.db.WithContext(ctx).Create(&model.LedgerTransactionModel{
		TransactionID: transaction.TransactionID,
		CreatedAt:     transaction.CreatedAt,
	}).Error; err != nil {
		return nil, mapError(err)
	}

	entryModels := make([]model.LedgerEntryModel, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		entryModels = append(entryModels, model.LedgerEntryModel{
			TransactionID: entry.TransactionID,
			AccountID:     entry.AccountID,
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt,
		})
	}
	if err := r.db.WithContext(ctx).Create(&entryModels).Error; err != nil {
		return nil, mapError(err)
	}

	entryIDs := make([]int64, 0, len(entryModels))
	for _, entryModel := range entryModels {
		entryIDs = append(entryIDs, entryModel.ID)
	}

	return entryIDs, nil
}

func (r *ledgerTransactionAggregateRepoImpl) persistEvents(ctx context.Context, events []eventpkg.Event) error {
	for _, evt := range events {
		eventModel, err := r.buildEventModel(evt)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := r.db.WithContext(ctx).Create(&eventModel).Error; err != nil {
			return mapError(err)
		}
	}

	return nil
}

func (r *ledgerTransactionAggregateRepoImpl) buildEventModel(evt eventpkg.Event) (model.LedgerEventModel, error) {
	data, err := r.serializer.Marshal(evt.EventData)
	if err != nil {
		return model.LedgerEventModel{}, stackErr.Error(fmt.Errorf("marshal ledger event data failed: %v", err))
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return model.LedgerEventModel{
		ID:            uuid.NewString(),
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}, nil
}

func newLedgerTransactionSerializer() eventpkg.Serializer {
	serializer := eventpkg.NewSerializer()
	if err := serializer.RegisterAggregate(&ledgeraggregate.LedgerTransactionAggregate{}); err != nil {
		panic(fmt.Sprintf("register ledger transaction aggregate serializer failed: %v", err))
	}
	return serializer
}
