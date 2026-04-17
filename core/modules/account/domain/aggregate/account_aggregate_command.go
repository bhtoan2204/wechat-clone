package aggregate

import (
	"go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

func NewAccountAggregate(accountID string) (*AccountAggregate, error) {
	agg := &AccountAggregate{}
	if err := event.InitAggregate(&agg.AggregateRoot, agg, accountID); err != nil {
		return nil, stackErr.Error(err)
	}

	return agg, nil
}
