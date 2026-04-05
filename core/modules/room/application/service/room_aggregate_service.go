package service

import (
	"context"
	"time"

	"go-socket/core/modules/room/domain/aggregate"
	"go-socket/core/modules/room/domain/repos"
	roomtypes "go-socket/core/modules/room/types"
	eventpkg "go-socket/core/shared/pkg/event"
)

type RoomAggregateService struct{}

func NewRoomAggregateService() *RoomAggregateService {
	return &RoomAggregateService{}
}

func (s *RoomAggregateService) PublishRoomCreated(ctx context.Context, outboxRepo repos.RoomOutboxEventsRepository, roomID string, roomType roomtypes.RoomType, memberCount int) error {
	roomAggregate, err := aggregate.NewRoomAggregate(roomID)
	if err != nil {
		return err
	}
	if err := roomAggregate.RecordRoomCreated(roomType, memberCount); err != nil {
		return err
	}
	return eventpkg.NewPublisher(outboxRepo).PublishAggregate(ctx, roomAggregate)
}

func (s *RoomAggregateService) PublishMemberAdded(ctx context.Context, outboxRepo repos.RoomOutboxEventsRepository, roomID, memberID string, memberRole roomtypes.RoomRole, joinedAt time.Time) error {
	roomAggregate, err := aggregate.NewRoomAggregate(roomID)
	if err != nil {
		return err
	}
	if err := roomAggregate.RecordMemberAdded(memberID, memberRole, joinedAt); err != nil {
		return err
	}
	return eventpkg.NewPublisher(outboxRepo).PublishAggregate(ctx, roomAggregate)
}

func (s *RoomAggregateService) PublishMemberRemoved(ctx context.Context, outboxRepo repos.RoomOutboxEventsRepository, roomID, memberID string, memberRole roomtypes.RoomRole, removedAt time.Time) error {
	roomAggregate, err := aggregate.NewRoomAggregate(roomID)
	if err != nil {
		return err
	}
	if err := roomAggregate.RecordMemberRemoved(memberID, memberRole, removedAt); err != nil {
		return err
	}
	return eventpkg.NewPublisher(outboxRepo).PublishAggregate(ctx, roomAggregate)
}

func (s *RoomAggregateService) PublishMessageCreated(ctx context.Context, outboxRepo repos.RoomOutboxEventsRepository, roomID, messageID, senderID, senderName, senderEmail, content string, sentAt time.Time) error {
	roomAggregate, err := aggregate.NewRoomAggregate(roomID)
	if err != nil {
		return err
	}
	if err := roomAggregate.RecordMessageCreated(messageID, senderID, senderName, senderEmail, content, sentAt); err != nil {
		return err
	}
	return eventpkg.NewPublisher(outboxRepo).PublishAggregate(ctx, roomAggregate)
}
