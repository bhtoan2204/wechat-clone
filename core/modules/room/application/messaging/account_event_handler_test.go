package messaging

import (
	"context"
	"testing"

	"go-socket/core/modules/room/domain/entity"
	sharedevents "go-socket/core/shared/contracts/events"
)

type roomAccountProjectionRepoStub struct {
	projected *entity.AccountEntity
}

func (s *roomAccountProjectionRepoStub) ProjectAccount(_ context.Context, account *entity.AccountEntity) error {
	s.projected = account
	return nil
}

func (s *roomAccountProjectionRepoStub) ListByAccountIDs(context.Context, []string) ([]*entity.AccountEntity, error) {
	return nil, nil
}

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
	repo := &roomAccountProjectionRepoStub{}
	handler := &messageHandler{accountRepo: repo}

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

	if err := handler.handleAccountEvent(context.Background(), raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if repo.projected == nil {
		t.Fatalf("expected projected account to be saved")
	}
	if repo.projected.AccountID != "acc-3" {
		t.Fatalf("expected account_id acc-3, got %s", repo.projected.AccountID)
	}
	if repo.projected.DisplayName != "Alice Updated" {
		t.Fatalf("expected display name Alice Updated, got %s", repo.projected.DisplayName)
	}
	if repo.projected.Username != "alice" {
		t.Fatalf("expected username alice, got %s", repo.projected.Username)
	}
	if repo.projected.AvatarObjectKey != "avatars/alice.png" {
		t.Fatalf("expected avatar avatars/alice.png, got %s", repo.projected.AvatarObjectKey)
	}
}
