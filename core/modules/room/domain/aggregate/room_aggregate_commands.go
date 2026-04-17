package aggregate

import (
	"go-socket/core/modules/room/domain/valueobject"
	roomtypes "go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
	"time"
)

func NewRoomAggregate(roomID string) (*RoomAggregate, error) {
	agg := &RoomAggregate{}
	if err := event.InitAggregate(&agg.AggregateRoot, agg, roomID); err != nil {
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
	joinedAt, err := normalizeRoomEventOccurredAt(joinedAt)
	if err != nil {
		return stackErr.Error(err)
	}

	return r.ApplyChange(r, &EventRoomMemberAdded{
		RoomID:         r.AggregateID(),
		MemberID:       memberID,
		MemberRole:     memberRole,
		MemberJoinedAt: joinedAt,
	})
}

func (r *RoomAggregate) RecordMemberRemoved(memberID string, memberRole roomtypes.RoomRole, removedAt time.Time) error {
	removedAt, err := normalizeRoomEventOccurredAt(removedAt)
	if err != nil {
		return stackErr.Error(err)
	}

	return r.ApplyChange(r, &EventRoomMemberRemoved{
		RoomID:         r.AggregateID(),
		MemberID:       memberID,
		MemberRole:     memberRole,
		MemberJoinedAt: removedAt,
	})
}

func (r *RoomAggregate) RecordMessageCreated(
	messageID string,
	content string,
	messageType string,
	sentAt time.Time,
	sender valueobject.Sender,
	reference valueobject.MessageReference,
	mentions valueobject.MessageMentions,
	attachment *valueobject.FileAttachment,
) error {
	sentAt, err := normalizeRoomEventOccurredAt(sentAt)
	if err != nil {
		return stackErr.Error(err)
	}

	var normalizedAttachment valueobject.FileAttachment
	if attachment != nil {
		normalizedAttachment = *attachment
	}

	return r.ApplyChange(r, &EventRoomMessageCreated{
		RoomID:                 r.AggregateID(),
		RoomName:               r.RoomName,
		RoomType:               r.RoomType.String(),
		MessageID:              messageID,
		MessageContent:         content,
		MessageType:            messageType,
		ReplyToMessageID:       reference.ReplyToMessageID,
		ForwardedFromMessageID: reference.ForwardedFromMessageID,
		FileName:               normalizedAttachment.FileName,
		FileSize:               normalizedAttachment.FileSize,
		MimeType:               normalizedAttachment.MimeType,
		ObjectKey:              normalizedAttachment.ObjectKey,
		MessageSenderID:        sender.ID,
		MessageSenderName:      sender.Name,
		MessageSenderEmail:     sender.Email,
		MessageSentAt:          sentAt,
		Mentions:               mentions.Items,
		MentionAll:             mentions.MentionAll,
		MentionedAccountIDs:    mentions.AccountIDs,
	})
}

func normalizeRoomEventOccurredAt(value time.Time) (time.Time, error) {
	if value.IsZero() {
		return time.Time{}, ErrRoomEventOccurredAtRequired
	}
	return value.UTC(), nil
}
