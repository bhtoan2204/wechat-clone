package aggregate

import (
	"testing"
	"time"

	"go-socket/core/modules/notification/types"
)

func TestNotificationAggregateCreateAndMarkRead(t *testing.T) {
	createdAt := time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)
	readAt := createdAt.Add(5 * time.Minute)

	agg, err := NewNotificationAggregate("notif-1")
	if err != nil {
		t.Fatalf("NewNotificationAggregate() error = %v", err)
	}
	if err := agg.Create("acc-1", types.NotificationTypeRoomMention, "Subject", "Body", createdAt); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	snapshot, err := agg.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if snapshot.AccountID != "acc-1" || snapshot.Type != types.NotificationTypeRoomMention {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
	if snapshot.IsRead {
		t.Fatalf("expected unread notification")
	}

	changed, err := agg.MarkRead(readAt)
	if err != nil {
		t.Fatalf("MarkRead() error = %v", err)
	}
	if !changed {
		t.Fatalf("expected MarkRead() to change state")
	}

	snapshot, err = agg.Snapshot()
	if err != nil {
		t.Fatalf("Snapshot() error = %v", err)
	}
	if !snapshot.IsRead || snapshot.ReadAt == nil || !snapshot.ReadAt.Equal(readAt) {
		t.Fatalf("expected read notification at %v, got %+v", readAt, snapshot)
	}
}
