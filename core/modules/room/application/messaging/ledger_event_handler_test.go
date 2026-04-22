package messaging

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	out "wechat-clone/core/modules/room/application/dto/out"
	"wechat-clone/core/modules/room/application/service"
	"wechat-clone/core/modules/room/domain/aggregate"
	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/modules/room/domain/repos"
	roomtypes "wechat-clone/core/modules/room/types"
	sharedevents "wechat-clone/core/shared/contracts/events"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func TestDecodeLedgerAccountTransferPayloadDerivesParticipantsFromEntrySign(t *testing.T) {
	raw, err := json.Marshal(sharedevents.LedgerAccountTransferredToAccountEvent{
		TransactionID: "txn-1",
		ToAccountID:   "receiver-1",
		Currency:      "vnd",
		Amount:        2500,
		BookedAt:      time.Date(2026, 4, 19, 8, 30, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	ctx := context.WithValue(context.Background(), ledgerTransferSenderAccountIDKey{}, "sender-1")
	payload, err := decodeLedgerAccountTransferPayload(ctx, raw)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if payload.SenderAccountID != "sender-1" {
		t.Fatalf("expected sender sender-1, got %s", payload.SenderAccountID)
	}
	if payload.ReceiverAccountID != "receiver-1" {
		t.Fatalf("expected receiver receiver-1, got %s", payload.ReceiverAccountID)
	}
	if payload.AmountMinor != 2500 {
		t.Fatalf("expected amount_minor 2500, got %d", payload.AmountMinor)
	}
	if payload.Currency != "VND" {
		t.Fatalf("expected currency VND, got %s", payload.Currency)
	}
}

func TestDecodeLedgerAccountTransferPayloadRejectsUnbalancedEntries(t *testing.T) {
	raw, err := json.Marshal(sharedevents.LedgerAccountTransferredToAccountEvent{
		TransactionID: "txn-2",
		ToAccountID:   "receiver-1",
		Currency:      "USD",
		Amount:        0,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	ctx := context.WithValue(context.Background(), ledgerTransferSenderAccountIDKey{}, "sender-1")
	if _, err := decodeLedgerAccountTransferPayload(ctx, raw); err == nil {
		t.Fatal("expected invalid ledger transfer payload to fail")
	}
}

func TestHandleLedgerAccountTransferredToAccountSkipsDuplicateTransferMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseRepo := repos.NewMockRepos(ctrl)
	messageRepo := repos.NewMockMessageRepository(ctrl)

	handler := &messageHandler{
		baseRepo: baseRepo,
	}

	raw, err := json.Marshal(sharedevents.LedgerAccountTransferredToAccountEvent{
		TransactionID: "txn-dup",
		ToAccountID:   "receiver-1",
		Currency:      "VND",
		Amount:        2500,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	ctx := context.WithValue(context.Background(), ledgerTransferSenderAccountIDKey{}, "sender-1")
	baseRepo.EXPECT().MessageRepository().Return(messageRepo).Times(1)
	messageRepo.EXPECT().
		GetMessageByID(gomock.Any(), transferMessageID("txn-dup")).
		Return(&entity.MessageEntity{ID: transferMessageID("txn-dup")}, nil).
		Times(1)

	if err := handler.handleLedgerAccountTransferredToAccount(ctx, raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandleLedgerAccountTransferredToAccountCreatesDeterministicTransferMessage(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseRepo := repos.NewMockRepos(ctrl)
	txRepos := repos.NewMockRepos(ctrl)
	txRoomAggRepo := repos.NewMockRoomAggregateRepository(ctrl)
	messageRepo := repos.NewMockMessageRepository(ctrl)
	realtime := service.NewMockService(ctrl)

	roomAgg := mustBuildDirectRoomAggregate(t, "room-1", "sender-1", "receiver-1")

	handler := &messageHandler{
		baseRepo: baseRepo,
		svc:      realtime,
	}

	createdAt := time.Date(2026, 4, 19, 8, 30, 0, 0, time.UTC)
	raw, err := json.Marshal(sharedevents.LedgerAccountTransferredToAccountEvent{
		TransactionID: "txn-123",
		ToAccountID:   "receiver-1",
		Currency:      "VND",
		Amount:        2500,
		BookedAt:      createdAt,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	ctx := context.WithValue(context.Background(), ledgerTransferSenderAccountIDKey{}, "sender-1")
	baseRepo.EXPECT().MessageRepository().Return(messageRepo).Times(1)
	messageRepo.EXPECT().
		GetMessageByID(gomock.Any(), transferMessageID("txn-123")).
		Return(nil, nil).
		Times(1)
	baseRepo.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(repos.Repos) error) error {
			return fn(txRepos)
		}).
		Times(1)
	txRepos.EXPECT().RoomAggregateRepository().Return(txRoomAggRepo).Times(2)
	txRoomAggRepo.EXPECT().
		LoadByDirectKey(gomock.Any(), entity.CanonicalDirectKey("sender-1", "receiver-1")).
		Return(roomAgg, nil).
		Times(1)
	txRoomAggRepo.EXPECT().
		Save(gomock.Any(), roomAgg).
		Return(nil).
		Times(1)
	realtime.EXPECT().
		EmitMessage(gomock.Any(), gomock.AssignableToTypeOf(roomtypes.MessagePayload{})).
		DoAndReturn(func(_ context.Context, payload roomtypes.MessagePayload) error {
			out, ok := payload.Payload.(*out.ChatMessageResponse)
			if !ok {
				t.Fatalf("expected chat message response payload, got %T", payload.Payload)
			}
			if out.ID != transferMessageID("txn-123") {
				t.Fatalf("expected deterministic message id, got %s", out.ID)
			}
			if out.Message != "VND 2500" {
				t.Fatalf("expected exact transfer message body, got %q", out.Message)
			}
			if out.SenderID != "sender-1" {
				t.Fatalf("expected sender sender-1, got %s", out.SenderID)
			}
			if out.CreatedAt != createdAt.Format(time.RFC3339) {
				t.Fatalf("expected created_at %s, got %s", createdAt.Format(time.RFC3339), out.CreatedAt)
			}
			return nil
		}).
		Times(1)

	if err := handler.handleLedgerAccountTransferredToAccount(ctx, raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestHandleLedgerAccountTransferredToAccountCreatesDirectRoomWhenMissing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	baseRepo := repos.NewMockRepos(ctrl)
	txRepos := repos.NewMockRepos(ctrl)
	txRoomAggRepo := repos.NewMockRoomAggregateRepository(ctrl)
	messageRepo := repos.NewMockMessageRepository(ctrl)
	realtime := service.NewMockService(ctrl)

	handler := &messageHandler{
		baseRepo: baseRepo,
		svc:      realtime,
	}

	createdAt := time.Date(2026, 4, 19, 8, 30, 0, 0, time.UTC)
	raw, err := json.Marshal(sharedevents.LedgerAccountTransferredToAccountEvent{
		TransactionID: "txn-create-room",
		ToAccountID:   "receiver-1",
		Currency:      "VND",
		Amount:        2500,
		BookedAt:      createdAt,
	})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	ctx := context.WithValue(context.Background(), ledgerTransferSenderAccountIDKey{}, "sender-1")
	baseRepo.EXPECT().MessageRepository().Return(messageRepo).Times(1)
	messageRepo.EXPECT().
		GetMessageByID(gomock.Any(), transferMessageID("txn-create-room")).
		Return(nil, nil).
		Times(1)
	baseRepo.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(repos.Repos) error) error {
			return fn(txRepos)
		}).
		Times(1)
	txRepos.EXPECT().RoomAggregateRepository().Return(txRoomAggRepo).Times(2)
	txRoomAggRepo.EXPECT().
		LoadByDirectKey(gomock.Any(), entity.CanonicalDirectKey("sender-1", "receiver-1")).
		Return(nil, stackNotFound()).
		Times(1)
	txRoomAggRepo.EXPECT().
		Save(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, agg *aggregate.RoomStateAggregate) error {
			if agg == nil || agg.Room() == nil {
				t.Fatal("expected room aggregate to be created")
			}
			if agg.Room().DirectKey != entity.CanonicalDirectKey("sender-1", "receiver-1") {
				t.Fatalf("expected direct room key, got %s", agg.Room().DirectKey)
			}
			return nil
		}).
		Times(1)
	realtime.EXPECT().
		EmitMessage(gomock.Any(), gomock.AssignableToTypeOf(roomtypes.MessagePayload{})).
		Return(nil).
		Times(1)

	if err := handler.handleLedgerAccountTransferredToAccount(ctx, raw); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func stackNotFound() error {
	return gorm.ErrRecordNotFound
}

func mustBuildDirectRoomAggregate(t *testing.T, roomID, senderID, receiverID string) *aggregate.RoomStateAggregate {
	t.Helper()

	now := time.Date(2026, 4, 19, 8, 0, 0, 0, time.UTC)
	room, err := entity.NewDirectConversationRoom(roomID, senderID, receiverID, now)
	if err != nil {
		t.Fatalf("build room: %v", err)
	}

	ownerMember, err := entity.NewRoomMember("member-1", room.ID, senderID, roomtypes.RoomRoleOwner, now)
	if err != nil {
		t.Fatalf("build owner member: %v", err)
	}
	receiverMember, err := entity.NewRoomMember("member-2", room.ID, receiverID, roomtypes.RoomRoleMember, now)
	if err != nil {
		t.Fatalf("build receiver member: %v", err)
	}

	agg, err := aggregate.RestoreRoomStateAggregate(room, []*entity.RoomMemberEntity{ownerMember, receiverMember}, 1)
	if err != nil {
		t.Fatalf("build room aggregate: %v", err)
	}

	return agg
}
