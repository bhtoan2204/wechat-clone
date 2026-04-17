package aggregate

import (
	"errors"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/event"
	"strings"
	"time"
)

var (
	ErrRoomAggregateIDMismatch     = errors.New("room id mismatch")
	ErrRoomEventOccurredAtRequired = errors.New("room event occurred_at is required")
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
		return r.applyRoomCreated(e.AggregateID, data)
	case *EventRoomMemberAdded:
		return r.applyRoomMemberAdded(data)
	case *EventRoomMemberRemoved:
		return r.applyRoomMemberRemoved(data)
	case *EventRoomMessageCreated:
		return r.applyRoomMessageCreated(data)
	default:
		return event.ErrUnsupportedEventType
	}
}

func (r *RoomAggregate) applyRoomCreated(
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

func (r *RoomAggregate) applyRoomMemberAdded(
	data *EventRoomMemberAdded,
) error {
	if err := r.ensureRoomID(data.RoomID); err != nil {
		return err
	}

	r.MemberCount++
	return nil
}

func (r *RoomAggregate) applyRoomMemberRemoved(
	data *EventRoomMemberRemoved,
) error {
	if err := r.ensureRoomID(data.RoomID); err != nil {
		return err
	}

	if r.MemberCount > 0 {
		r.MemberCount--
	}
	return nil
}

func (r *RoomAggregate) applyRoomMessageCreated(
	data *EventRoomMessageCreated,
) error {
	if err := r.ensureRoomID(data.RoomID); err != nil {
		return err
	}

	r.LastMessageID = data.MessageID
	r.LastMessageAt = data.MessageSentAt
	r.LastMessageContent = data.MessageContent
	r.LastMessageSenderID = data.MessageSenderID

	return nil
}

func (r *RoomAggregate) ensureRoomID(roomID string) error {
	roomID = strings.TrimSpace(roomID)
	if roomID == "" {
		return entity.ErrRoomIDRequired
	}
	if r.RoomID == "" {
		r.RoomID = roomID
		return nil
	}
	if r.RoomID != roomID {
		return ErrRoomAggregateIDMismatch
	}
	return nil
}
