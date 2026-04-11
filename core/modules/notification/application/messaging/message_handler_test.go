package messaging

import (
	"context"
	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/modules/notification/domain/entity"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/utils"
	"testing"
)

func TestDecodeAccountCreatedPayloadObject(t *testing.T) {
	raw := []byte(`{"AccountID":"acc-1","Email":"a@example.com","CreatedAt":"2026-03-03T13:05:32.218937909+07:00"}`)

	payloadAny, err := decodeEventPayload(context.Background(), sharedevents.EventAccountCreated, raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	payload, ok := payloadAny.(*sharedevents.AccountCreatedEvent)
	if !ok {
		t.Fatalf("expected AccountCreatedEvent, got %T", payloadAny)
	}
	if payload.AccountID != "acc-1" {
		t.Fatalf("expected account_id acc-1, got %s", payload.AccountID)
	}
	if payload.Email != "a@example.com" {
		t.Fatalf("expected email a@example.com, got %s", payload.Email)
	}
}

func TestDecodeAccountCreatedPayloadEncodedString(t *testing.T) {
	raw := []byte(`"{\"AccountID\":\"acc-2\",\"Email\":\"b@example.com\",\"CreatedAt\":\"2026-03-03T13:05:32.218937909+07:00\"}"`)

	payloadAny, err := decodeEventPayload(context.Background(), sharedevents.EventAccountCreated, raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	payload, ok := payloadAny.(*sharedevents.AccountCreatedEvent)
	if !ok {
		t.Fatalf("expected AccountCreatedEvent, got %T", payloadAny)
	}
	if payload.AccountID != "acc-2" {
		t.Fatalf("expected account_id acc-2, got %s", payload.AccountID)
	}
	if payload.Email != "b@example.com" {
		t.Fatalf("expected email b@example.com, got %s", payload.Email)
	}
}

func TestDecodeAccountCreatedPayloadEmpty(t *testing.T) {
	_, err := decodeEventPayload(context.Background(), sharedevents.EventAccountCreated, []byte(`""`))
	if err == nil {
		t.Fatalf("expected error when event_data is empty")
	}
}

type emailSenderStub struct {
	to      string
	subject string
	body    string
	called  bool
}

func (s *emailSenderStub) Send(_ context.Context, to, subject, body string) error {
	s.called = true
	s.to = to
	s.subject = subject
	s.body = body
	return nil
}

type notificationRepoStub struct {
	created []*entity.NotificationEntity
}

func (s *notificationRepoStub) CreateNotification(_ context.Context, notification *entity.NotificationEntity) error {
	s.created = append(s.created, notification)
	return nil
}

func (s *notificationRepoStub) ListNotifications(context.Context, utils.QueryOptions) ([]*out.NotificationResponse, error) {
	return nil, nil
}

func TestHandleAccountEventWithLowercaseFields(t *testing.T) {
	stub := &emailSenderStub{}
	repo := &notificationRepoStub{}
	handler := &messageHandler{emailSender: stub, notificationRepo: repo}

	raw := []byte(`{
		"id": 1,
		"aggregate_id": "acc-2",
		"aggregate_type": "account",
		"version": 1,
		"event_name": "EventAccountCreated",
		"event_data": {"AccountID":"acc-2","Email":"b@example.com","CreatedAt":"2026-03-03T13:05:32.218937909+07:00"},
		"created_at": "2026-03-03T13:05:32.218937909+07:00"
	}`)

	if err := handler.handleAccountEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !stub.called {
		t.Fatalf("expected email sender to be called")
	}
	if stub.to != "b@example.com" {
		t.Fatalf("expected email recipient b@example.com, got %s", stub.to)
	}
	if stub.subject != "Welcome to Go Socket" {
		t.Fatalf("expected welcome subject, got %s", stub.subject)
	}
	if len(repo.created) != 1 {
		t.Fatalf("expected notification to be created")
	}
}

func TestHandleRoomOutboxEventCreatesMentionNotifications(t *testing.T) {
	repo := &notificationRepoStub{}
	handler := &messageHandler{notificationRepo: repo}

	raw := []byte(`{
		"id": 11,
		"aggregate_id": "room-1",
		"aggregate_type": "RoomAggregate",
		"version": 3,
		"event_name": "EventRoomMessageCreated",
		"event_data": {
			"room_id": "room-1",
			"room_name": "Backend",
			"room_type": "group",
			"message_id": "msg-1",
			"message_content": "hello team",
			"message_type": "text",
			"message_sender_id": "acc-1",
			"message_sender_name": "Alice",
			"message_sent_at": "2026-04-12T10:00:00Z",
			"mention_all": false,
			"mentioned_account_ids": ["acc-2", "acc-3", "acc-2"]
		},
		"metadata": "{}",
		"created_at": "2026-04-12T10:00:00Z"
	}`)

	if err := handler.handleRoomOutboxEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(repo.created) != 2 {
		t.Fatalf("expected 2 mention notifications, got %d", len(repo.created))
	}
	if repo.created[0].Type != "room.mention" {
		t.Fatalf("expected room mention notification type, got %s", repo.created[0].Type)
	}
	if repo.created[0].Subject != "Alice mentioned you in Backend" {
		t.Fatalf("unexpected subject %q", repo.created[0].Subject)
	}
	if repo.created[0].Body != "hello team" {
		t.Fatalf("unexpected body %q", repo.created[0].Body)
	}
}
