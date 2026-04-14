package repos

import (
	"context"
	"go-socket/core/modules/notification/domain/aggregate"
	"go-socket/core/modules/notification/domain/entity"
)

//go:generate mockgen -package=repos -destination=push_subscription_repo_mock.go -source=push_subscription_repo.go
type PushSubscriptionRepository interface {
	LoadByAccountAndEndpoint(ctx context.Context, accountID, endpoint string) (*aggregate.PushSubscriptionAggregate, error)
	Save(ctx context.Context, subscription *aggregate.PushSubscriptionAggregate) error
	ListPushSubscriptionsByAccountID(ctx context.Context, accountID string) ([]*entity.PushSubscription, error)
}
