package processor

import (
	"context"
	"encoding/json"
	roomprojection "go-socket/core/modules/room/application/projection"
	"strings"
	"testing"
)

type servingProjectorStub struct {
	roomSync    *roomprojection.RoomAggregateSync
	messageSync *roomprojection.MessageAggregateSync
	deletedRoom string
}

func (s *servingProjectorStub) SyncRoomAggregate(_ context.Context, projection *roomprojection.RoomAggregateSync) error {
	s.roomSync = projection
	return nil
}

func (s *servingProjectorStub) DeleteRoomAggregate(_ context.Context, roomID string) error {
	s.deletedRoom = roomID
	return nil
}

func (s *servingProjectorStub) SyncMessageAggregate(_ context.Context, projection *roomprojection.MessageAggregateSync) error {
	s.messageSync = projection
	return nil
}

type searchIndexerStub struct {
	message *roomprojection.MessageProjection
	roomID  string
}

func (s *searchIndexerStub) SyncMessage(_ context.Context, message *roomprojection.MessageProjection) error {
	s.message = message
	return nil
}

func (s *searchIndexerStub) DeleteRoom(_ context.Context, roomID string) error {
	s.roomID = roomID
	return nil
}

func TestHandleRoomOutboxEventProjectsServingAndSearchSnapshots(t *testing.T) {
	serving := &servingProjectorStub{}
	search := &searchIndexerStub{}
	processor := &processor{
		servingProjector: serving,
		searchIndexer:    search,
	}

	raw := []byte(`{
		"id": 31,
		"aggregate_id": "room-1",
		"aggregate_type": "RoomAggregate",
		"version": 4,
		"event_name": "EventMessageAggregateProjectionSynced",
		"event_data": {
			"message": {
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
			"receipts": [{
				"room_id": "room-1",
				"message_id": "msg-1",
				"account_id": "acc-2",
				"status": "sent",
				"created_at": "2026-04-12T12:00:00Z",
				"updated_at": "2026-04-12T12:00:00Z"
			}]
		},
		"metadata": "{}",
		"created_at": "2026-04-12T12:00:00Z"
	}`)

	if err := processor.handleRoomOutboxEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if serving.messageSync == nil {
		t.Fatalf("expected serving snapshot to be written")
	}
	if serving.messageSync.Message == nil || serving.messageSync.Message.RoomID != "room-1" || serving.messageSync.Message.MessageID != "msg-1" {
		t.Fatalf("unexpected message projection %+v", serving.messageSync.Message)
	}
	if !serving.messageSync.Message.MentionAll {
		t.Fatalf("expected mention_all to be projected")
	}
	if len(serving.messageSync.Message.MentionedAccountIDs) != 3 {
		t.Fatalf("expected mentioned_account_ids to be preserved, got %+v", serving.messageSync.Message.MentionedAccountIDs)
	}
	if len(serving.messageSync.Receipts) != 1 || serving.messageSync.Receipts[0].AccountID != "acc-2" {
		t.Fatalf("expected receipt snapshot to be projected, got %+v", serving.messageSync.Receipts)
	}

	if search.message == nil {
		t.Fatalf("expected search message to be indexed")
	}
	if search.message.RoomName != "Backend" {
		t.Fatalf("expected room_name Backend, got %q", search.message.RoomName)
	}
	if search.message.ReplyToMessageID != "msg-0" {
		t.Fatalf("expected reply_to_message_id msg-0, got %q", search.message.ReplyToMessageID)
	}
}

func TestMessageProjectionJSONUsesSnakeCaseFields(t *testing.T) {
	document := &roomprojection.MessageProjection{
		RoomID:            "room-1",
		MessageID:         "msg-1",
		MessageSenderID:   "acc-1",
		MessageSenderName: "Alice",
		Mentions: []roomprojection.ProjectionMention{
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
