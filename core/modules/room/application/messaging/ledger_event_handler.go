package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	reflect "reflect"
	"strings"
	"time"

	roomsupport "go-socket/core/modules/room/application/support"
	"go-socket/core/modules/room/domain/aggregate"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/types"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

func (h *messageHandler) handleLedgerAccountTransferredToAccount(ctx context.Context, raw json.RawMessage) error {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventLedgerAccountTransferredToAccount, raw)
	if err != nil {
		return stackErr.Error(fmt.Errorf("decode event payload failed: %v", err))
	}

	payload, ok := payloadAny.(*sharedevents.LedgerTransaction)
	if !ok {
		return stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventLedgerAccountTransferredToAccount))
	}
	if payload == nil || len(payload.Entries) != 2 {
		return stackErr.Error(fmt.Errorf("ledger transfer payload must contain exactly 2 entries"))
	}

	senderID := strings.TrimSpace(payload.Entries[0].AccountID) // sender is always the debit entry
	receiverID := strings.TrimSpace(payload.Entries[1].AccountID)
	if senderID == "" || receiverID == "" {
		return stackErr.Error(fmt.Errorf("ledger transfer payload account ids are required"))
	}

	roomAgg, err := h.baseRepo.RoomAggregateRepository().LoadByDirectKey(ctx, entity.CanonicalDirectKey(senderID, receiverID))
	if err != nil {
		return stackErr.Error(fmt.Errorf("load transfer room failed: %v", err))
	}
	if roomAgg == nil || roomAgg.Room() == nil {
		return stackErr.Error(fmt.Errorf("direct room not found for transfer participants"))
	}

	now := time.Now().UTC()
	message, err := roomAgg.SendMessage(
		uuid.NewString(),
		senderID,
		entity.MessageParams{
			Message:     fmt.Sprintf("%f", math.Abs(float64(payload.Entries[0].Amount))),
			MessageType: entity.MessageTypeTransfer,
		},
		aggregate.MessageSenderIdentity{},
		aggregate.MessageOutboxPayload{},
		now,
	)
	if err != nil {
		return stackErr.Error(err)
	}

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, roomAgg))
	}); err != nil {
		return stackErr.Error(err)
	}

	msg, err := roomsupport.BuildMessageResultFromState(ctx, h.baseRepo, senderID, message)
	if err != nil {
		return stackErr.Error(err)
	}
	out := roomsupport.ToMessageResponse(msg)
	h.svc.EmitMessage(ctx, types.MessagePayload{
		RoomId:  out.RoomID,
		Type:    reflect.TypeOf(out).Elem().Name(),
		Payload: out,
	})

	return nil
}
