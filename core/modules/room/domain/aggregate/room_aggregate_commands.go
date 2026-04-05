package aggregate

import (
	roomtypes "go-socket/core/modules/room/types"
	"time"
)

func NewRoomAggregate(roomID string) (*RoomAggregate, error) {
	agg := &RoomAggregate{}
	agg.SetAggregateType("RoomAggregate")
	if err := agg.SetID(roomID); err != nil {
		return nil, err
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

func (r *RoomAggregate) RecordMessageCreated(messageID, senderID, senderName, senderEmail, content string, sentAt time.Time) error {
	return r.ApplyChange(r, &EventRoomMessageCreated{
		RoomID:             r.AggregateID(),
		MessageID:          messageID,
		MessageContent:     content,
		MessageSenderID:    senderID,
		MessageSenderName:  senderName,
		MessageSenderEmail: senderEmail,
		MessageSentAt:      sentAt,
	})
}
