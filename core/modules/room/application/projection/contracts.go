package projection

import (
	"context"

	sharedevents "wechat-clone/core/shared/contracts/events"
)

const (
	EventRoomAggregateProjectionSynced    = sharedevents.EventRoomAggregateProjectionSynced
	EventRoomAggregateProjectionDeleted   = sharedevents.EventRoomAggregateProjectionDeleted
	EventMessageAggregateProjectionSynced = sharedevents.EventMessageAggregateProjectionSynced
)

//go:generate mockgen -package=projection -destination=contracts_mock.go -source=contracts.go
type ServingProjector interface {
	SyncRoomAggregate(ctx context.Context, projection *RoomAggregateSync) error
	DeleteRoomAggregate(ctx context.Context, roomID string) error
	SyncMessageAggregate(ctx context.Context, projection *MessageAggregateSync) error
}

//go:generate mockgen -package=projection -destination=contracts_mock.go -source=contracts.go
type MessageSearchIndexer interface {
	SyncMessage(ctx context.Context, message *MessageProjection) error
	DeleteRoom(ctx context.Context, roomID string) error
}

type ProjectionMention = sharedevents.RoomProjectionMention
type ProjectionReaction = sharedevents.RoomProjectionReaction
type RoomAggregateDeleted = sharedevents.RoomAggregateProjectionDeletedEvent
type RoomAggregateSync = sharedevents.RoomAggregateProjectionSyncedEvent
type RoomProjection = sharedevents.RoomProjection
type RoomLastMessageProjection = sharedevents.RoomLastMessageProjection
type RoomMemberProjection = sharedevents.RoomMemberProjection
type MessageAggregateSync = sharedevents.RoomMessageAggregateSyncedEvent
type MessageProjection = sharedevents.RoomMessageProjection
type MessageReceiptProjection = sharedevents.RoomMessageReceiptProjection
type MessageDeletionProjection = sharedevents.RoomMessageDeletionProjection
