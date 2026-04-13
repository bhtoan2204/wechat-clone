package processor

import (
	"context"
	"encoding/json"
	"go-socket/core/shared/contracts/events"
	"strings"
	"testing"
)

type timelineProjectorStub struct {
	projected *events.TimelineMessageProjection
}

func (s *timelineProjectorStub) ProjectRoom(_ context.Context, _ *events.RoomProjection) error {
	return nil
}

func (s *timelineProjectorStub) DeleteProjectedRoom(_ context.Context, _ string) error {
	return nil
}

func (s *timelineProjectorStub) ProjectRoomMember(_ context.Context, _ *events.RoomMemberProjection) error {
	return nil
}

func (s *timelineProjectorStub) DeleteProjectedRoomMember(_ context.Context, _, _ string) error {
	return nil
}

func (s *timelineProjectorStub) ProjectMessage(_ context.Context, projection *events.TimelineMessageProjection) error {
	s.projected = projection
	return nil
}

func (s *timelineProjectorStub) ProjectMessageReceipt(_ context.Context, _ *events.MessageReceiptProjection) error {
	return nil
}

func (s *timelineProjectorStub) ProjectMessageDeletion(_ context.Context, _ *events.MessageDeletionProjection) error {
	return nil
}

type searchIndexerStub struct {
	document *events.SearchMessageDocument
}

func (s *searchIndexerStub) UpsertMessage(_ context.Context, document *events.SearchMessageDocument) error {
	s.document = document
	return nil
}

func (p *searchIndexerStub) DeleteProjectedRoom(ctx context.Context, roomID string) error {
	panic("unimplemented")
}

func (p *searchIndexerStub) DeleteProjectedRoomMember(ctx context.Context, roomID string, accountID string) error {
	panic("unimplemented")
}

func (p *searchIndexerStub) ProjectMessage(ctx context.Context, projection *events.TimelineMessageProjection) error {
	panic("unimplemented")
}

func (p *searchIndexerStub) ProjectMessageDeletion(ctx context.Context, projection *events.MessageDeletionProjection) error {
	panic("unimplemented")
}

func (p *searchIndexerStub) ProjectMessageReceipt(ctx context.Context, projection *events.MessageReceiptProjection) error {
	panic("unimplemented")
}

func (p *searchIndexerStub) ProjectRoom(ctx context.Context, projection *events.RoomProjection) error {
	panic("unimplemented")
}

func (p *searchIndexerStub) ProjectRoomMember(ctx context.Context, projection *events.RoomMemberProjection) error {
	panic("unimplemented")
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
		"event_name": "EventRoomMessageProjectionUpserted",
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

func TestSearchMessageDocumentJSONUsesSnakeCaseFields(t *testing.T) {
	document := &events.SearchMessageDocument{
		RoomID:            "room-1",
		MessageID:         "msg-1",
		MessageSenderID:   "acc-1",
		MessageSenderName: "Alice",
		Mentions: []events.ProjectionMention{
			{
				AccountID:   "acc-2",
				DisplayName: "Bob",
				Username:    "bob",
			},
		},
	}

	payload, err := json.Marshal(document)
	if err != nil {
		t.Fatalf("expected marshal success, got %v", err)
	}

	jsonText := string(payload)
	for _, expected := range []string{
		`"room_id":"room-1"`,
		`"message_id":"msg-1"`,
		`"message_sender_id":"acc-1"`,
		`"message_sender_name":"Alice"`,
		`"account_id":"acc-2"`,
		`"display_name":"Bob"`,
		`"username":"bob"`,
	} {
		if !strings.Contains(jsonText, expected) {
			t.Fatalf("expected json payload to contain %s, got %s", expected, jsonText)
		}
	}

	for _, unexpected := range []string{`"RoomID"`, `"MessageID"`, `"MessageSenderID"`, `"AccountID"`} {
		if strings.Contains(jsonText, unexpected) {
			t.Fatalf("expected json payload to avoid Go field name %s, got %s", unexpected, jsonText)
		}
	}
}
