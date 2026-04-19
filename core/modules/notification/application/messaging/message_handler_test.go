package messaging

import (
	"context"
	"testing"
	"time"

	"wechat-clone/core/modules/notification/domain/aggregate"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
	sharedevents "wechat-clone/core/shared/contracts/events"

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
}

func TestHandleAccountEventSkipsExistingWelcomeNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	existingAgg, err := aggregate.NewNotificationAggregate(aggregate.WelcomeNotificationID("acc-3"))
	if err != nil {
		t.Fatalf("NewNotificationAggregate() error = %v", err)
	}
	if err := existingAgg.Create("acc-3", "account.created", "Welcome to Go Socket", "Welcome c@example.com!", time.Date(2026, 3, 3, 6, 5, 32, 0, time.UTC)); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	repo := notificationrepos.NewMockNotificationRepository(ctrl)
	repo.EXPECT().
		Load(gomock.Any(), aggregate.WelcomeNotificationID("acc-3")).
		Return(existingAgg, nil)

	baseRepo := notificationrepos.NewMockRepos(ctrl)
	baseRepo.EXPECT().NotificationRepository().Return(repo).AnyTimes()

	handler := &messageHandler{
		baseRepo: baseRepo,
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

	if err := handler.handleAccountEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
