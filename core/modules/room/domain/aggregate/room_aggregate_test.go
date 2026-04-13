package aggregate

import (
	"testing"
	"time"

	roomtypes "go-socket/core/modules/room/types"
)

func TestRoomAggregateRecordMemberAddedBindsRoomID(t *testing.T) {
	agg, err := NewRoomAggregate("room-1")
	if err != nil {
		t.Fatalf("NewRoomAggregate() error = %v", err)
	}

	if err := agg.RecordMemberAdded("member-1", roomtypes.RoomRoleMember, time.Now().UTC()); err != nil {
		t.Fatalf("RecordMemberAdded() error = %v", err)
	}

	if agg.RoomID != "room-1" {
		t.Fatalf("RoomID = %q, want %q", agg.RoomID, "room-1")
	}
	if agg.MemberCount != 1 {
		t.Fatalf("MemberCount = %d, want 1", agg.MemberCount)
	}
}

func TestRoomAggregateRecordMessageCreatedBindsRoomID(t *testing.T) {
	// agg, err := NewRoomAggregate("room-1")
	// if err != nil {
	// 	t.Fatalf("NewRoomAggregate() error = %v", err)
	// }

	// sentAt := time.Now().UTC()
	// if err := agg.RecordMessageCreated(
	// 	"Backend",
	// 	string(roomtypes.RoomTypeGroup),
	// 	"msg-1",
	// 	"sender-1",
	// 	"Alice",
	// 	"alice@example.com",
	// 	"hello",
	// 	"text",
	// 	"",
	// 	"",
	// 	"",
	// 	"",
	// 	"",
	// 	0,
	// 	sentAt,
	// 	[]sharedevents.RoomMessageMention{{AccountID: "member-2", DisplayName: "Bob"}},
	// 	false,
	// 	[]string{"member-2"},
	// ); err != nil {
	// 	t.Fatalf("RecordMessageCreated() error = %v", err)
	// }

	// if agg.RoomID != "room-1" {
	// 	t.Fatalf("RoomID = %q, want %q", agg.RoomID, "room-1")
	// }
	// if agg.LastMessageID != "msg-1" {
	// 	t.Fatalf("LastMessageID = %q, want %q", agg.LastMessageID, "msg-1")
	// }
	// if !agg.LastMessageAt.Equal(sentAt) {
	// 	t.Fatalf("LastMessageAt = %v, want %v", agg.LastMessageAt, sentAt)
	// }

	// events := agg.Events()
	// if len(events) != 1 {
	// 	t.Fatalf("expected 1 unsaved event, got %d", len(events))
	// }
	// data, ok := events[0].EventData.(*EventRoomMessageCreated)
	// if !ok {
	// 	t.Fatalf("expected EventRoomMessageCreated, got %T", events[0].EventData)
	// }
	// if len(data.MentionedAccountIDs) != 1 || data.MentionedAccountIDs[0] != "member-2" {
	// 	t.Fatalf("expected mentioned_account_ids to be propagated, got %+v", data.MentionedAccountIDs)
	// }
}
