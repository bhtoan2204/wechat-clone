package aggregate

import (
	"go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/event"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RoomAggregate struct {
	event.AggregateRoot

	RoomID              string
	RoomName            string
	RoomType            types.RoomType
	MemberCount         int
	LastMessageID       string
	LastMessageAt       time.Time
	LastMessageContent  string
	LastMessageSenderID string
}

func (r *RoomAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(
		&EventRoomCreated{},
		&EventRoomMemberAdded{},
		&EventRoomMemberRemoved{},
		&EventRoomMessageCreated{},
	)
}

func (r *RoomAggregate) Transition(e event.Event) error {
	switch data := e.EventData.(type) {
	case *EventRoomCreated:
		return r.onRoomCreated(e.AggregateID, data)
	case *EventRoomMemberAdded:
		return r.onRoomMemberAdded(data)
	case *EventRoomMemberRemoved:
		return r.onRoomMemberRemoved(data)
	case *EventRoomMessageCreated:
		return r.onRoomMessageCreated(data)
	default:
		return status.Error(codes.InvalidArgument, "unsupported event type")
	}
}

func (r *RoomAggregate) onRoomCreated(
	aggregateID string,
	data *EventRoomCreated,
) error {
	r.RoomID = aggregateID
	r.RoomType = data.RoomType
	r.MemberCount = data.MemberCount
	r.LastMessageID = data.LastMessageID
	r.LastMessageAt = data.LastMessageAt
	r.LastMessageContent = data.LastMessageContent
	r.LastMessageSenderID = data.LastMessageSenderID
	return nil
}

func (r *RoomAggregate) onRoomMemberAdded(
	data *EventRoomMemberAdded,
) error {
	if err := r.ensureRoomID(data.RoomID); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	r.MemberCount++
	return nil
}

func (r *RoomAggregate) onRoomMemberRemoved(
	data *EventRoomMemberRemoved,
) error {
	if err := r.ensureRoomID(data.RoomID); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	if r.MemberCount > 0 {
		r.MemberCount--
	}
	return nil
}

func (r *RoomAggregate) onRoomMessageCreated(
	data *EventRoomMessageCreated,
) error {
	if err := r.ensureRoomID(data.RoomID); err != nil {
		return status.Error(codes.InvalidArgument, err.Error())
	}

	r.LastMessageID = data.MessageID
	r.LastMessageAt = data.MessageSentAt
	r.LastMessageContent = data.MessageContent
	r.LastMessageSenderID = data.MessageSenderID

	return nil
}

func (r *RoomAggregate) ensureRoomID(roomID string) error {
	if roomID == "" {
		return status.Error(codes.InvalidArgument, "room id is required")
	}
	if r.RoomID == "" {
		r.RoomID = roomID
		return nil
	}
	if r.RoomID != roomID {
		return status.Error(codes.InvalidArgument, "room id mismatch")
	}
	return nil
}
