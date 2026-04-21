package repository

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
	"wechat-clone/core/modules/ledger/domain/eventstore"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"
)

type ledgerAccountAggregateRepositoryImpl struct {
	store eventstore.AggregateStore
}

func NewLedgerAccountAggregateRepoImpl(dbTX dbTX) ledgerrepos.LedgerAccountAggregateRepository {
	return newLedgerAccountAggregateRepoImpl(dbTX)
}

func newLedgerAccountAggregateRepoImpl(dbTX dbTX) *ledgerAccountAggregateRepositoryImpl {
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
