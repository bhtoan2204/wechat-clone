package service

import (
	"context"
	"time"

	"go-socket/core/modules/room/domain/aggregate"
	"go-socket/core/modules/room/domain/repos"
	roomtypes "go-socket/core/modules/room/types"
	sharedevents "go-socket/core/shared/contracts/events"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

type RoomAggregateService struct{}

func NewRoomAggregateService() *RoomAggregateService {
	return &RoomAggregateService{}
}

func (s *RoomAggregateService) PublishRoomCreated(ctx context.Context, outboxRepo repos.RoomOutboxEventsRepository, roomID string, roomType roomtypes.RoomType, memberCount int) error {
	roomAggregate, err := aggregate.NewRoomAggregate(roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := roomAggregate.RecordRoomCreated(roomType, memberCount); err != nil {
		return stackErr.Error(err)
	}
	return eventpkg.NewPublisher(outboxRepo).PublishAggregate(ctx, roomAggregate)
}

func (s *RoomAggregateService) PublishMemberAdded(ctx context.Context, outboxRepo repos.RoomOutboxEventsRepository, roomID, memberID string, memberRole roomtypes.RoomRole, joinedAt time.Time) error {
	roomAggregate, err := aggregate.NewRoomAggregate(roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := roomAggregate.RecordMemberAdded(memberID, memberRole, joinedAt); err != nil {
		return stackErr.Error(err)
	}
	return eventpkg.NewPublisher(outboxRepo).PublishAggregate(ctx, roomAggregate)
}

func (s *RoomAggregateService) PublishMemberRemoved(ctx context.Context, outboxRepo repos.RoomOutboxEventsRepository, roomID, memberID string, memberRole roomtypes.RoomRole, removedAt time.Time) error {
	roomAggregate, err := aggregate.NewRoomAggregate(roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := roomAggregate.RecordMemberRemoved(memberID, memberRole, removedAt); err != nil {
		return stackErr.Error(err)
	}
	return eventpkg.NewPublisher(outboxRepo).PublishAggregate(ctx, roomAggregate)
}

func (s *RoomAggregateService) PublishMessageCreated(
	ctx context.Context,
	outboxRepo repos.RoomOutboxEventsRepository,
	roomID,
	roomName,
	roomType,
	messageID,
	senderID,
	senderName,
	senderEmail,
	content,
	messageType,
	replyToMessageID,
	forwardedFromMessageID,
	fileName,
	mimeType,
	objectKey string,
	fileSize int64,
	sentAt time.Time,
	mentions []sharedevents.RoomMessageMention,
	mentionAll bool,
	mentionedAccountIDs []string,
) error {
	roomAggregate, err := aggregate.NewRoomAggregate(roomID)
	if err != nil {
		return stackErr.Error(err)
	}
	if err := roomAggregate.RecordMessageCreated(
		roomName,
		roomType,
		messageID,
		senderID,
		senderName,
		senderEmail,
		content,
		messageType,
		replyToMessageID,
		forwardedFromMessageID,
		fileName,
		mimeType,
		objectKey,
		fileSize,
		sentAt,
		mentions,
		mentionAll,
		mentionedAccountIDs,
	); err != nil {
		return stackErr.Error(err)
	}
	return eventpkg.NewPublisher(outboxRepo).PublishAggregate(ctx, roomAggregate)
}
