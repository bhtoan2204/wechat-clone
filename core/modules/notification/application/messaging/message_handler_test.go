package messaging

import (
	"context"
	"go-socket/core/modules/notification/application/adapter"
	"go-socket/core/modules/notification/domain/aggregate"
	"go-socket/core/modules/notification/domain/repos"
	"go-socket/core/modules/notification/types"
	sharedevents "go-socket/core/shared/contracts/events"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
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

func TestHandleAccountEventWithLowercaseFields(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	emailSender := adapter.NewMockEmailSender(ctrl)
	notificationRepo := repos.NewMockNotificationRepository(ctrl)

	handler := &messageHandler{
		emailSender:      emailSender,
		notificationRepo: notificationRepo,
	}

	raw := []byte(`{
		"id": 1,
		"aggregate_id": "acc-2",
		"aggregate_type": "account",
		"version": 1,
		"event_name": "EventAccountCreated",
		"event_data": {"AccountID":"acc-2","Email":"b@example.com","CreatedAt":"2026-03-03T13:05:32.218937909+07:00"},
		"created_at": "2026-03-03T13:05:32.218937909+07:00"
	}`)

	emailSender.EXPECT().
		Send(gomock.Any(), "b@example.com", "Welcome to Go Socket", gomock.Any()).
		Return(nil).
		Times(1)

	notificationRepo.EXPECT().
		Load(gomock.Any(), aggregate.WelcomeNotificationID("acc-2")).
		Return(nil, repos.ErrNotificationNotFound).
		Times(1)
	notificationRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.NotificationAggregate{})).
		DoAndReturn(func(_ context.Context, n *aggregate.NotificationAggregate) error {
			snapshot, err := n.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if snapshot == nil || snapshot.ID != aggregate.WelcomeNotificationID("acc-2") {
				t.Fatalf("expected welcome notification aggregate, got %+v", snapshot)
			}
			return nil
		}).
		Times(1)

	if err := handler.handleAccountEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandleRoomOutboxEventCreatesMentionNotifications(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	notificationRepo := repos.NewMockNotificationRepository(ctrl)

	handler := &messageHandler{
		notificationRepo: notificationRepo,
	}

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

	callCount := 0
	notificationRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.NotificationAggregate{})).
		DoAndReturn(func(_ context.Context, n *aggregate.NotificationAggregate) error {
			callCount++

			snapshot, err := n.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if snapshot.Type != "room.mention" {
				t.Fatalf("expected room.mention, got %s", snapshot.Type)
			}
			if snapshot.Subject != "Alice mentioned you in Backend" {
				t.Fatalf("unexpected subject %q", snapshot.Subject)
			}
			if snapshot.Body != "hello team" {
				t.Fatalf("unexpected body %q", snapshot.Body)
			}
			return nil
		}).
		Times(2)

	if err := handler.handleRoomOutboxEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if callCount != 2 {
		t.Fatalf("expected 2 notifications, got %d", callCount)
	}
}

func TestHandleAccountEventSkipsExistingWelcomeNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	emailSender := adapter.NewMockEmailSender(ctrl)
	notificationRepo := repos.NewMockNotificationRepository(ctrl)

	existingAgg, err := aggregate.NewNotificationAggregate(aggregate.WelcomeNotificationID("acc-3"))
	if err != nil {
		t.Fatalf("NewNotificationAggregate() error = %v", err)
	}
	if err := existingAgg.Create("acc-3", types.NotificationTypeAccountCreated, "Welcome to Go Socket", "Welcome c@example.com!", time.Date(2026, 3, 3, 6, 5, 32, 0, time.UTC)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	handler := &messageHandler{
		emailSender:      emailSender,
		notificationRepo: notificationRepo,
	}

	raw := []byte(`{
		"id": 1,
		"aggregate_id": "acc-3",
		"aggregate_type": "account",
		"version": 1,
		"event_name": "EventAccountCreated",
		"event_data": {"AccountID":"acc-3","Email":"c@example.com","CreatedAt":"2026-03-03T06:05:32Z"},
		"created_at": "2026-03-03T06:05:32Z"
	}`)

	notificationRepo.EXPECT().
		Load(gomock.Any(), aggregate.WelcomeNotificationID("acc-3")).
		Return(existingAgg, nil).
		Times(1)

	if err := handler.handleAccountEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
