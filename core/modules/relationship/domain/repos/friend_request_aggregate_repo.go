package repos

import (
	"context"
	"wechat-clone/core/modules/relationship/domain/aggregate"
)

type FriendRequestAggregateRepository interface {
	Load(ctx context.Context, friendRequestID string) (*aggregate.FriendRequestAggregate, error)
	LoadPendingByUsers(ctx context.Context, requesterID, addresseeID string) (*aggregate.FriendRequestAggregate, error)
	LoadPendingBetween(ctx context.Context, userA, userB string) (*aggregate.FriendRequestAggregate, error)
	Save(ctx context.Context, agg *aggregate.FriendRequestAggregate) error
}
