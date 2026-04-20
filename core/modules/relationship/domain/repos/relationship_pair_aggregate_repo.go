package repos

import (
	"context"

	"wechat-clone/core/modules/relationship/domain/aggregate"
)

type RelationshipPairAggregateRepository interface {
	LoadForUpdate(ctx context.Context, actorID, targetID string) (*aggregate.RelationshipPairAggregate, error)
	Save(ctx context.Context, agg *aggregate.RelationshipPairAggregate) error
}
