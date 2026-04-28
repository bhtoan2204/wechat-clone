package messaging

import (
	"context"
	"testing"
	"time"

	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/modules/room/domain/repos"
	sharedevents "wechat-clone/core/shared/contracts/events"

	"go.uber.org/mock/gomock"
)

func TestDecodeAccountCreatedPayloadUsesSharedContract(t *testing.T) {
	raw := []byte(`{"AccountID":"acc-1","Email":"a@example.com","DisplayName":"Alice","CreatedAt":"2026-03-03T13:05:32.218937909+07:00"}`)

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
	if payload.DisplayName != "Alice" {
		t.Fatalf("expected display name Alice, got %s", payload.DisplayName)
	}
}

func TestHandleAccountEventProfileUpdatedProjectsUsernameAndAvatar(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accountRepo := repos.NewMockRoomAccountRepository(ctrl)
	handler := &messageHandler{accountRepo: accountRepo}

	raw := []byte(`{
		"id": 1,
		"aggregate_id": "acc-3",
		"aggregate_type": "account",
		"version": 2,
		"event_name": "EventAccountProfileUpdated",
		"event_data": {
			"AccountID":"acc-3",
			"DisplayName":"Alice Updated",
			"Username":"alice",
			"AvatarObjectKey":"avatars/alice.png",
			"UpdatedAt":"2026-03-03T13:05:32.218937909+07:00"
		},
		"created_at": "2026-03-03T13:05:32.218937909+07:00"
	}`)

	accountRepo.EXPECT().
		ProjectAccount(gomock.Any(), gomock.AssignableToTypeOf(&entity.AccountEntity{})).
		DoAndReturn(func(_ context.Context, account *entity.AccountEntity) error {
			if account == nil {
				t.Fatalf("expected projected account")
			}
			if account.AccountID != "acc-3" {
				t.Fatalf("expected account_id acc-3, got %s", account.AccountID)
			}
			if account.DisplayName != "Alice Updated" {
				t.Fatalf("expected display name Alice Updated, got %s", account.DisplayName)
			}
			if account.Username != "alice" {
				t.Fatalf("expected username alice, got %s", account.Username)
			}
			if account.AvatarObjectKey != "avatars/alice.png" {
				t.Fatalf("expected avatar avatars/alice.png, got %s", account.AvatarObjectKey)
			}
			return nil
		}).
		Times(1)

	if err := handler.handleAccountEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandleAccountEventCreatedFallsBackToEmailWhenDisplayNameMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accountRepo := repos.NewMockRoomAccountRepository(ctrl)
	handler := &messageHandler{accountRepo: accountRepo}

	raw := []byte(`{
		"id": 22,
		"aggregate_id": "acc-legacy",
		"aggregate_type": "AccountAggregate",
		"version": 1,
		"event_name": "EventAccountCreated",
		"event_data": "{\"AccountID\":\"acc-legacy\",\"Email\":\"legacy@example.com\",\"CreatedAt\":\"2026-04-06T02:16:22.067488606+07:00\"}",
		"created_at": "2026-04-05T19:16:22.000000Z"
	}`)

	accountRepo.EXPECT().
		ProjectAccount(gomock.Any(), gomock.AssignableToTypeOf(&entity.AccountEntity{})).
		DoAndReturn(func(_ context.Context, account *entity.AccountEntity) error {
			if account == nil {
				t.Fatalf("expected projected account")
			}
			if account.DisplayName != "legacy@example.com" {
				t.Fatalf("expected display name fallback to email, got %q", account.DisplayName)
			}
			if account.UpdatedAt.IsZero() {
				t.Fatalf("expected updated_at to be populated")
			}
			return nil
		}).
		Times(1)

	if err := handler.handleAccountEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestResolveAccountCreatedDisplayNameFallsBackInPriorityOrder(t *testing.T) {
	now := time.Now().UTC()

	tests := []struct {
		name    string
		payload *sharedevents.AccountCreatedEvent
		want    string
	}{
		{
			name: "uses display name when present",
			payload: &sharedevents.AccountCreatedEvent{
				AccountID:   "acc-1",
				Email:       "user@example.com",
				DisplayName: "Alice",
				CreatedAt:   now,
			},
			want: "Alice",
		},
		{
			name: "falls back to email for legacy payload",
			payload: &sharedevents.AccountCreatedEvent{
				AccountID: "acc-2",
				Email:     "legacy@example.com",
				CreatedAt: now,
			},
			want: "legacy@example.com",
		},
		{
			name: "falls back to account id when email missing",
			payload: &sharedevents.AccountCreatedEvent{
				AccountID: "acc-3",
				CreatedAt: now,
			},
			want: "acc-3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveAccountCreatedDisplayName(tt.payload); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
