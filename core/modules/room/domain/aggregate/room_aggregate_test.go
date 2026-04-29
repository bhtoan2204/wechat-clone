package aggregate

import (
	"testing"
	"time"

	"wechat-clone/core/modules/room/domain/valueobject"
	roomtypes "wechat-clone/core/modules/room/types"
)

func TestRoomAggregateRecordMemberAddedBindsRoomID(t *testing.T) {
	agg, err := NewRoomAggregate("room-1")
	if err != nil {
		t.Fatalf("NewRoomAggregate() error = %v", err)
	}

	joinedAt := time.Now().UTC()
	if err := agg.RecordMemberAdded("member-1", roomtypes.RoomRoleMember, joinedAt); err != nil {
		t.Fatalf("RecordMemberAdded() error = %v", err)
	}

	if agg.AggregateID() != "room-1" {
		t.Fatalf("AggregateID() = %q, want %q", agg.AggregateID(), "room-1")
	}

	events := agg.CloneEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(events))
	}
	data, ok := events[0].EventData.(*EventRoomMemberAdded)
	if !ok {
		t.Fatalf("expected EventRoomMemberAdded, got %T", events[0].EventData)
	}
	if data.RoomID != "room-1" || data.MemberID != "member-1" {
		t.Fatalf("unexpected member-added event %+v", data)
	}
	if events[0].CreatedAt != joinedAt.UnixMilli() {
		t.Fatalf("CreatedAt = %d, want %d", events[0].CreatedAt, joinedAt.UnixMilli())
	}
}

func TestRoomAggregateRecordMessageCreatedBindsRoomID(t *testing.T) {
	agg, err := NewRoomAggregate("room-1")
	if err != nil {
		t.Fatalf("NewRoomAggregate() error = %v", err)
	}

	sentAt := time.Now().UTC()
	if err := agg.RecordMessageCreated(
		"msg-1",
		"hello",
		"text",
		sentAt,
		valueobject.Sender{ID: "sender-1", Name: "Alice", Email: "alice@example.com"},
		valueobject.MessageReference{},
		valueobject.MessageMentions{AccountIDs: []string{"member-2"}},
		nil,
	); err != nil {
		t.Fatalf("RecordMessageCreated() error = %v", err)
	}

	if agg.AggregateID() != "room-1" {
		t.Fatalf("AggregateID() = %q, want %q", agg.AggregateID(), "room-1")
	}

	events := agg.CloneEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(events))
	}
	data, ok := events[0].EventData.(*EventRoomMessageCreated)
	if !ok {
		t.Fatalf("expected EventRoomMessageCreated, got %T", events[0].EventData)
	}
	if data.RoomID != "room-1" || data.MessageID != "msg-1" {
		t.Fatalf("unexpected message-created event %+v", data)
	}
	if len(data.MentionedAccountIDs) != 1 || data.MentionedAccountIDs[0] != "member-2" {
		t.Fatalf("expected mentioned_account_ids to be propagated, got %+v", data.MentionedAccountIDs)
	}
}
