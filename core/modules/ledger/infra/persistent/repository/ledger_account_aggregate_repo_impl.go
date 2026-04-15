package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	ledgeraggregate "go-socket/core/modules/ledger/domain/aggregate"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

type ledgerAccountAggregateRepositoryImpl struct {
	store aggregateStore
}

func NewLedgerAccountAggregateRepoImpl(dbTX dbTX) ledgerrepos.LedgerAccountAggregateRepository {
	serializer := eventpkg.NewSerializer()
	if err := serializer.RegisterAggregate(&ledgeraggregate.LedgerAccountAggregate{}); err != nil {
		panic(fmt.Sprintf("register ledger account aggregate serializer failed: %v", err))
	}

	return &ledgerAccountAggregateRepositoryImpl{
		store: newAggregateStore(dbTX, serializer),
	}
}

func (r *ledgerAccountAggregateRepositoryImpl) Load(ctx context.Context, accountID string) (*ledgeraggregate.LedgerAccountAggregate, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return nil, stackErr.Error(fmt.Errorf("account id is required"))
	}

	agg, err := ledgeraggregate.NewLedgerAccountAggregate(accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := r.store.Get(ctx, accountID, agg); err != nil {
		if errors.Is(err, ledgerrepos.ErrNotFound) {
			return nil, nil
		}
		return nil, stackErr.Error(err)
	}

	return agg, nil
}

func (r *ledgerAccountAggregateRepositoryImpl) Save(ctx context.Context, aggregate *ledgeraggregate.LedgerAccountAggregate) error {
	if aggregate == nil {
		return stackErr.Error(fmt.Errorf("ledger account aggregate is nil"))
	}
	if reflect.ValueOf(aggregate).IsNil() {
		return stackErr.Error(fmt.Errorf("ledger account aggregate is nil"))
	}
	return stackErr.Error(r.store.Save(ctx, aggregate))
}
