package messaging

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	reflect "reflect"
	"time"

	roomsupport "wechat-clone/core/modules/room/application/support"
	"wechat-clone/core/modules/room/domain/aggregate"
	"wechat-clone/core/modules/room/domain/entity"
	"wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/modules/room/types"
	roomtypes "wechat-clone/core/modules/room/types"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func (h *messageHandler) handleLedgerAccountTransferredToAccount(ctx context.Context, raw json.RawMessage) error {
	transfer, err := decodeLedgerAccountTransferPayload(ctx, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode ledger transfer payload failed: %w", err))
	}

	messageID := transferMessageID(transfer.TransactionID)
	existingMessage, err := h.baseRepo.MessageRepository().GetMessageByID(ctx, messageID)
	if err != nil {
		return stackErr.Error(fmt.Errorf("load transfer message failed: %w", err))
	}
	if existingMessage != nil {
		return nil
	}

	now := transfer.CreatedAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	var message *entity.MessageEntity

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		roomAgg, err := ensureTransferDirectRoom(ctx, txRepos, transfer.SenderAccountID, transfer.ReceiverAccountID, now)
		if err != nil {
			return stackErr.Error(fmt.Errorf("load transfer room failed: %w", err))
		}

		message, err = roomAgg.SendMessage(
			messageID,
			transfer.SenderAccountID,
			entity.MessageParams{
				Message:     formatTransferMessageBody(transfer.Currency, transfer.AmountMinor),
				MessageType: entity.MessageTypeTransfer,
			},
			aggregate.MessageSenderIdentity{},
			aggregate.MessageOutboxPayload{},
			now,
		)
		if err != nil {
			return stackErr.Error(err)
		}

		return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, roomAgg))
	}); err != nil {
		return stackErr.Error(err)
	}

	msg, err := roomsupport.BuildMessageResultFromState(ctx, h.baseRepo, transfer.SenderAccountID, message)
	if err != nil {
		return stackErr.Error(err)
	}
	out := roomsupport.ToMessageResponse(msg)
	if err := h.svc.EmitMessage(ctx, types.MessagePayload{
		RoomId:  out.RoomID,
		Type:    reflect.TypeOf(out).Elem().Name(),
		Payload: out,
	}); err != nil {
		return stackErr.Error(fmt.Errorf("failed to emit realtime message after handling ledger transfer event: %w", err))
	}

	return nil
}

func ensureTransferDirectRoom(
	ctx context.Context,
	txRepos repos.Repos,
	senderAccountID string,
	receiverAccountID string,
	now time.Time,
) (*aggregate.RoomStateAggregate, error) {
	directKey := entity.CanonicalDirectKey(senderAccountID, receiverAccountID)
	roomAgg, err := txRepos.RoomAggregateRepository().LoadByDirectKey(ctx, directKey)
	if err == nil && roomAgg != nil && roomAgg.Room() != nil {
		return roomAgg, nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, stackErr.Error(err)
	}

	room, err := entity.NewDirectConversationRoom(uuid.NewString(), senderAccountID, receiverAccountID, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	ownerMember, err := entity.NewRoomMember(uuid.NewString(), room.ID, senderAccountID, roomtypes.RoomRoleOwner, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	receiverMember, err := entity.NewRoomMember(uuid.NewString(), room.ID, receiverAccountID, roomtypes.RoomRoleMember, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	roomAgg, err = aggregate.NewConversationRoomAggregate(
		room,
		[]*entity.RoomMemberEntity{ownerMember, receiverMember},
		"",
		"",
		now,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return roomAgg, nil
}
