package repos

import (
	"context"

	"wechat-clone/core/modules/room/domain/aggregate"
)

//go:generate mockgen -package=repos -destination=message_aggregate_repo_mock.go -source=message_aggregate_repo.go
type MessageAggregateRepository interface {
	Load(ctx context.Context, messageID string) (*aggregate.MessageStateAggregate, error)
	LoadForRecipient(ctx context.Context, messageID, recipientAccountID string) (*aggregate.MessageStateAggregate, error)
	Save(ctx context.Context, agg *aggregate.MessageStateAggregate) error
}
