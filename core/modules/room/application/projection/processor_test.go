package projection

import (
	"context"
	"testing"
)

type timelineProjectorStub struct {
	projected *TimelineMessageProjection
}

func (s *timelineProjectorStub) UpsertMessage(_ context.Context, projection *TimelineMessageProjection) error {
	s.projected = projection
	return nil
}

type searchIndexerStub struct {
	document *SearchMessageDocument
}

func (s *searchIndexerStub) UpsertMessage(_ context.Context, document *SearchMessageDocument) error {
	s.document = document
	return nil
}

func TestHandleRoomOutboxEventProjectsTimelineAndSearch(t *testing.T) {
	timeline := &timelineProjectorStub{}
	search := &searchIndexerStub{}
	processor := &processor{
		timelineProjector: timeline,
		searchIndexer:     search,
	}

	raw := []byte(`{
		"id": 31,
		"aggregate_id": "room-1",
		"aggregate_type": "RoomAggregate",
		"version": 4,
		"event_name": "EventRoomMessageCreated",
		"event_data": {
			"room_id": "room-1",
			"room_name": "Backend",
			"room_type": "group",
			"message_id": "msg-1",
			"message_content": "hello team",
			"message_type": "text",
			"reply_to_message_id": "msg-0",
			"file_name": "",
			"file_size": 0,
			"mime_type": "",
			"object_key": "",
			"message_sender_id": "acc-1",
			"message_sender_name": "Alice",
			"message_sent_at": "2026-04-12T12:00:00Z",
			"mentions": [{"account_id":"acc-2","display_name":"Bob","username":"bob"}],
			"mention_all": true,
			"mentioned_account_ids": ["acc-2", "acc-3", "acc-2"]
		},
		"metadata": "{}",
		"created_at": "2026-04-12T12:00:00Z"
	}`)

	if err := processor.handleRoomOutboxEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if timeline.projected == nil {
		t.Fatalf("expected timeline projection to be written")
	}
	if timeline.projected.RoomID != "room-1" || timeline.projected.MessageID != "msg-1" {
		t.Fatalf("unexpected timeline projection %+v", timeline.projected)
	}
	if !timeline.projected.MentionAll {
		t.Fatalf("expected mention_all to be projected")
	}
	if len(timeline.projected.MentionedAccountIDs) != 2 {
		t.Fatalf("expected unique mentioned_account_ids, got %+v", timeline.projected.MentionedAccountIDs)
	}

	if search.document == nil {
		t.Fatalf("expected search document to be indexed")
	}
	if search.document.RoomName != "Backend" {
		t.Fatalf("expected room_name Backend, got %q", search.document.RoomName)
	}
	if search.document.ReplyToMessageID != "msg-0" {
		t.Fatalf("expected reply_to_message_id msg-0, got %q", search.document.ReplyToMessageID)
	}
}
