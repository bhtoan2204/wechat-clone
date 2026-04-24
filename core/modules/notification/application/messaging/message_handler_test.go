package messaging

import (
	"context"
	"testing"
	"time"

	notificationservice "wechat-clone/core/modules/notification/application/service"
	"wechat-clone/core/modules/notification/domain/aggregate"
	notificationrepos "wechat-clone/core/modules/notification/domain/repos"
	notificationtypes "wechat-clone/core/modules/notification/types"
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

func TestHandleRelationshipEventCreatesFriendRequestNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := notificationrepos.NewMockNotificationRepository(ctrl)
	realtime := notificationservice.NewMockRealtimeService(ctrl)
	baseRepo := notificationrepos.NewMockRepos(ctrl)
	baseRepo.EXPECT().NotificationRepository().Return(repo).AnyTimes()

	accountID := "acc-target"
	requestID := "req-1"
	notificationID := aggregate.FriendRequestNotificationID(notificationtypes.NotificationTypeFriendRequestSent, requestID, accountID)

	repo.EXPECT().Load(gomock.Any(), notificationID).Return(nil, notificationrepos.ErrNotificationNotFound)
	repo.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.NotificationAggregate{})).DoAndReturn(func(_ context.Context, agg *aggregate.NotificationAggregate) error {
		snapshot, err := agg.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v", err)
		}
		if snapshot.AccountID != accountID {
			t.Fatalf("AccountID = %s, want %s", snapshot.AccountID, accountID)
		}
		if snapshot.Type != notificationtypes.NotificationTypeFriendRequestSent {
			t.Fatalf("Type = %s, want %s", snapshot.Type, notificationtypes.NotificationTypeFriendRequestSent)
		}
		return nil
	})
	repo.EXPECT().CountUnread(gomock.Any(), accountID).Return(3, nil)
	realtime.EXPECT().EmitMessage(gomock.Any(), gomock.Any()).Return(nil)

	handler := &messageHandler{
		baseRepo: baseRepo,
		realtime: realtime,
		push:     nil,
		email:    nil,
	}

	raw := []byte(`{
		"id": 11,
		"aggregate_id": "pair:acc-source:acc-target",
		"aggregate_type": "RelationshipPairAggregate",
		"version": 1,
		"event_name": "EventRelationshipPairFriendRequestSent",
		"event_data": {
			"RequestID":"req-1",
			"RequesterID":"acc-source",
			"AddresseeID":"acc-target",
			"CreatedAt":"2026-04-23T08:00:00Z"
		},
		"created_at": "2026-04-23T08:00:00Z"
	}`)

	if err := handler.handleRelationshipOutboxEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandleRelationshipEventCreatesAcceptedNotificationForRequester(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := notificationrepos.NewMockNotificationRepository(ctrl)
	realtime := notificationservice.NewMockRealtimeService(ctrl)
	baseRepo := notificationrepos.NewMockRepos(ctrl)
	baseRepo.EXPECT().NotificationRepository().Return(repo).AnyTimes()

	accountID := "acc-requester"
	requestID := "req-2"
	notificationID := aggregate.FriendRequestNotificationID(notificationtypes.NotificationTypeFriendRequestAccepted, requestID, accountID)

	repo.EXPECT().Load(gomock.Any(), notificationID).Return(nil, notificationrepos.ErrNotificationNotFound)
	repo.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.NotificationAggregate{})).DoAndReturn(func(_ context.Context, agg *aggregate.NotificationAggregate) error {
		snapshot, err := agg.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v", err)
		}
		if snapshot.AccountID != accountID {
			t.Fatalf("AccountID = %s, want %s", snapshot.AccountID, accountID)
		}
		if snapshot.Type != notificationtypes.NotificationTypeFriendRequestAccepted {
			t.Fatalf("Type = %s, want %s", snapshot.Type, notificationtypes.NotificationTypeFriendRequestAccepted)
		}
		return nil
	})
	repo.EXPECT().CountUnread(gomock.Any(), accountID).Return(5, nil)
	realtime.EXPECT().EmitMessage(gomock.Any(), gomock.Any()).Return(nil)

	handler := &messageHandler{
		baseRepo: baseRepo,
		realtime: realtime,
	}

	raw := []byte(`{
		"id": 12,
		"aggregate_id": "pair:acc-requester:acc-addressee",
		"aggregate_type": "RelationshipPairAggregate",
		"version": 2,
		"event_name": "EventRelationshipPairFriendRequestAccepted",
		"event_data": {
			"RequestID":"req-2",
			"RequesterID":"acc-requester",
			"AddresseeID":"acc-addressee",
			"CreatedAt":"2026-04-23T08:00:00Z",
			"FriendshipID":"friendship-1",
			"AcceptedAt":"2026-04-23T08:05:00Z"
		},
		"created_at": "2026-04-23T08:05:00Z"
	}`)

	if err := handler.handleRelationshipOutboxEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestDecodeRelationshipPayloadObject(t *testing.T) {
	raw := []byte(`{"RequestID":"req-9","RequesterID":"acc-a","AddresseeID":"acc-b","CreatedAt":"2026-04-23T08:00:00Z"}`)

	payloadAny, err := decodeEventPayload(context.Background(), sharedevents.EventRelationshipPairFriendRequestSent, raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if payloadAny == nil {
		t.Fatal("expected payload, got nil")
	}
}

func TestHandlePaymentEventCreatesWithdrawalSuccessNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := notificationrepos.NewMockNotificationRepository(ctrl)
	realtime := notificationservice.NewMockRealtimeService(ctrl)
	baseRepo := notificationrepos.NewMockRepos(ctrl)
	baseRepo.EXPECT().NotificationRepository().Return(repo).AnyTimes()

	accountID := "acc-withdraw"
	paymentID := "pay-withdraw-1"
	notificationID := aggregate.PaymentNotificationID(notificationtypes.NotificationTypeWithdrawalSucceeded, paymentID, accountID)

	repo.EXPECT().Load(gomock.Any(), notificationID).Return(nil, notificationrepos.ErrNotificationNotFound)
	repo.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.NotificationAggregate{})).DoAndReturn(func(_ context.Context, agg *aggregate.NotificationAggregate) error {
		snapshot, err := agg.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v", err)
		}
		if snapshot.AccountID != accountID {
			t.Fatalf("AccountID = %s, want %s", snapshot.AccountID, accountID)
		}
		if snapshot.Type != notificationtypes.NotificationTypeWithdrawalSucceeded {
			t.Fatalf("Type = %s, want %s", snapshot.Type, notificationtypes.NotificationTypeWithdrawalSucceeded)
		}
		return nil
	})
	repo.EXPECT().CountUnread(gomock.Any(), accountID).Return(1, nil)
	realtime.EXPECT().EmitMessage(gomock.Any(), gomock.Any()).Return(nil)

	handler := &messageHandler{
		baseRepo: baseRepo,
		realtime: realtime,
	}

	raw := []byte(`{
		"id": 21,
		"aggregate_id": "pay-withdraw-1",
		"aggregate_type": "PaymentIntentAggregate",
		"version": 2,
		"event_name": "PaymentSucceededEvent",
		"event_data": {
			"workflow":"WITHDRAWAL",
			"payment_id":"pay-withdraw-1",
			"transaction_id":"txn-withdraw-1",
			"debit_account_id":"acc-withdraw",
			"amount":100,
			"currency":"VND",
			"succeeded_at":"2026-04-24T10:00:00Z"
		},
		"created_at": "2026-04-24T10:00:00Z"
	}`)

	if err := handler.handlePaymentOutboxEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandlePaymentEventCreatesWithdrawalRequestedNotification(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	repo := notificationrepos.NewMockNotificationRepository(ctrl)
	realtime := notificationservice.NewMockRealtimeService(ctrl)
	baseRepo := notificationrepos.NewMockRepos(ctrl)
	baseRepo.EXPECT().NotificationRepository().Return(repo).AnyTimes()

	accountID := "acc-withdraw"
	paymentID := "pay-withdraw-req-1"
	notificationID := aggregate.PaymentNotificationID(notificationtypes.NotificationTypeWithdrawalRequested, paymentID, accountID)

	repo.EXPECT().Load(gomock.Any(), notificationID).Return(nil, notificationrepos.ErrNotificationNotFound)
	repo.EXPECT().Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.NotificationAggregate{})).DoAndReturn(func(_ context.Context, agg *aggregate.NotificationAggregate) error {
		snapshot, err := agg.Snapshot()
		if err != nil {
			t.Fatalf("Snapshot() error = %v", err)
		}
		if snapshot.AccountID != accountID {
			t.Fatalf("AccountID = %s, want %s", snapshot.AccountID, accountID)
		}
		if snapshot.Type != notificationtypes.NotificationTypeWithdrawalRequested {
			t.Fatalf("Type = %s, want %s", snapshot.Type, notificationtypes.NotificationTypeWithdrawalRequested)
		}
		return nil
	})
	repo.EXPECT().CountUnread(gomock.Any(), accountID).Return(1, nil)
	realtime.EXPECT().EmitMessage(gomock.Any(), gomock.Any()).Return(nil)

	handler := &messageHandler{
		baseRepo: baseRepo,
		realtime: realtime,
	}

	raw := []byte(`{
		"id": 20,
		"aggregate_id": "pay-withdraw-req-1",
		"aggregate_type": "PaymentIntentAggregate",
		"version": 2,
		"event_name": "PaymentWithdrawalRequestedEvent",
		"event_data": {
			"payment_id":"pay-withdraw-req-1",
			"transaction_id":"txn-withdraw-req-1",
			"provider":"stripe",
			"debit_account_id":"acc-withdraw",
			"destination_account_id":"bank:dest-1",
			"amount":100,
			"fee_amount":5,
			"provider_amount":100,
			"currency":"VND",
			"status":"CREATING",
			"requested_at":"2026-04-24T10:00:00Z"
		},
		"created_at": "2026-04-24T10:00:00Z"
	}`)

	if err := handler.handlePaymentOutboxEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
