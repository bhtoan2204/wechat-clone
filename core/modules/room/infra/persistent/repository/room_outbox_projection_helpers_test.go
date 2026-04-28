package repository

import (
	"context"
	"testing"
	"time"

	"wechat-clone/core/modules/room/domain/entity"
)

type accountProjectionStoreFake struct {
	listByAccountIDs func(ctx context.Context, accountIDs []string) ([]*entity.AccountEntity, error)
}

func (f accountProjectionStoreFake) ProjectAccount(context.Context, *entity.AccountEntity) error {
	return nil
}

func (f accountProjectionStoreFake) ListByAccountIDs(ctx context.Context, accountIDs []string) ([]*entity.AccountEntity, error) {
	if f.listByAccountIDs == nil {
		return nil, nil
	}
	return f.listByAccountIDs(ctx, accountIDs)
}

func TestEnrichRoomMembersWithAccountProjectionsFillsProfileFields(t *testing.T) {
	t.Parallel()

	accountRepo := accountProjectionStoreFake{
		listByAccountIDs: func(_ context.Context, accountIDs []string) ([]*entity.AccountEntity, error) {
			if len(accountIDs) != 1 || accountIDs[0] != "acc-1" {
				t.Fatalf("expected account ids [acc-1], got %#v", accountIDs)
			}
			return []*entity.AccountEntity{
				{
					AccountID:       "acc-1",
					DisplayName:     "Alice",
					Username:        "alice",
					AvatarObjectKey: "avatars/alice.png",
				},
			}, nil
		},
	}

	now := time.Date(2026, time.April, 14, 2, 15, 0, 0, time.UTC)
	members := []*entity.RoomMemberEntity{
		{
			ID:        "member-1",
			RoomID:    "room-1",
			AccountID: "acc-1",
			CreatedAt: now,
			UpdatedAt: now,
		},
	}

	enriched, err := enrichRoomMembersWithAccountProjections(context.Background(), accountRepo, members)
	if err != nil {
		t.Fatalf("enrichRoomMembersWithAccountProjections() error = %v", err)
	}

	if len(enriched) != 1 {
		t.Fatalf("expected 1 enriched member, got %d", len(enriched))
	}
	if enriched[0].DisplayName != "Alice" {
		t.Fatalf("expected display name Alice, got %q", enriched[0].DisplayName)
	}
	if enriched[0].Username != "alice" {
		t.Fatalf("expected username alice, got %q", enriched[0].Username)
	}
	if enriched[0].AvatarObjectKey != "avatars/alice.png" {
		t.Fatalf("expected avatar key avatars/alice.png, got %q", enriched[0].AvatarObjectKey)
	}
	if members[0].DisplayName != "" {
		t.Fatalf("expected original member slice to remain unchanged, got %q", members[0].DisplayName)
	}
}

func TestMapRoomMemberProjectionsIncludesProfileFields(t *testing.T) {
	t.Parallel()

	projections := mapRoomMemberProjections([]*entity.RoomMemberEntity{
		{
			ID:              "member-1",
			RoomID:          "room-1",
			AccountID:       "acc-1",
			DisplayName:     "Alice",
			Username:        "alice",
			AvatarObjectKey: "avatars/alice.png",
		},
	})

	if len(projections) != 1 {
		t.Fatalf("expected 1 projection, got %d", len(projections))
	}
	if projections[0].DisplayName != "Alice" {
		t.Fatalf("expected display name Alice, got %q", projections[0].DisplayName)
	}
	if projections[0].Username != "alice" {
		t.Fatalf("expected username alice, got %q", projections[0].Username)
	}
	if projections[0].AvatarObjectKey != "avatars/alice.png" {
		t.Fatalf("expected avatar key avatars/alice.png, got %q", projections[0].AvatarObjectKey)
	}
}
