package messaging

import (
	"context"
	"go-socket/core/modules/account/domain/aggregate"
	"go-socket/core/modules/notification/application/dto/out"
	"go-socket/core/modules/notification/domain/entity"
	"go-socket/core/shared/utils"
	"testing"
)

func TestDecodeAccountCreatedPayloadObject(t *testing.T) {
	raw := []byte(`{"AccountID":"acc-1","Email":"a@example.com","CreatedAt":"2026-03-03T13:05:32.218937909+07:00"}`)

	payloadAny, err := decodeEventPayload(context.Background(), "EventAccountCreated", raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	payload, ok := payloadAny.(*aggregate.EventAccountCreated)
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

	payloadAny, err := decodeEventPayload(context.Background(), "EventAccountCreated", raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	payload, ok := payloadAny.(*aggregate.EventAccountCreated)
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
	_, err := decodeEventPayload(context.Background(), "EventAccountCreated", []byte(`""`))
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
	created *entity.NotificationEntity
}

func (s *notificationRepoStub) CreateNotification(_ context.Context, notification *entity.NotificationEntity) error {
	s.created = notification
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
	if repo.created == nil {
		t.Fatalf("expected notification to be created")
	}
}
