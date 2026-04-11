package aggregate

import (
	roomtypes "go-socket/core/modules/room/types"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/stackErr"
	"time"
)

func NewRoomAggregate(roomID string) (*RoomAggregate, error) {
	agg := &RoomAggregate{}
	agg.SetAggregateType("RoomAggregate")
	if err := agg.SetID(roomID); err != nil {
		return nil, stackErr.Error(err)
	}
	return agg, nil
}

func (r *RoomAggregate) RecordRoomCreated(roomType roomtypes.RoomType, memberCount int) error {
	return r.ApplyChange(r, &EventRoomCreated{
		RoomID:      r.AggregateID(),
		RoomType:    roomType,
		MemberCount: memberCount,
	})
}

func (r *RoomAggregate) RecordMemberAdded(memberID string, memberRole roomtypes.RoomRole, joinedAt time.Time) error {
	return r.ApplyChange(r, &EventRoomMemberAdded{
		RoomID:         r.AggregateID(),
		MemberID:       memberID,
		MemberRole:     memberRole,
		MemberJoinedAt: joinedAt,
	})
}

func (r *RoomAggregate) RecordMemberRemoved(memberID string, memberRole roomtypes.RoomRole, removedAt time.Time) error {
	return r.ApplyChange(r, &EventRoomMemberRemoved{
		RoomID:         r.AggregateID(),
		MemberID:       memberID,
		MemberRole:     memberRole,
		MemberJoinedAt: removedAt,
	})
}

func (r *RoomAggregate) RecordMessageCreated(
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
	return r.ApplyChange(r, &EventRoomMessageCreated{
		RoomID:                 r.AggregateID(),
		RoomName:               roomName,
		RoomType:               roomType,
		MessageID:              messageID,
		MessageContent:         content,
		MessageType:            messageType,
		ReplyToMessageID:       replyToMessageID,
		ForwardedFromMessageID: forwardedFromMessageID,
		FileName:               fileName,
		FileSize:               fileSize,
		MimeType:               mimeType,
		ObjectKey:              objectKey,
		MessageSenderID:        senderID,
		MessageSenderName:      senderName,
		MessageSenderEmail:     senderEmail,
		MessageSentAt:          sentAt,
		Mentions:               mentions,
		MentionAll:             mentionAll,
		MentionedAccountIDs:    mentionedAccountIDs,
	})
}
