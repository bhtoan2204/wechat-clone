package command

import (
	"context"
	"time"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomsupport "go-socket/core/modules/room/application/support"
	roomrepos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type pinChatMessageHandler struct {
	baseRepo roomrepos.Repos
}

func NewPinChatMessageHandler(baseRepo roomrepos.Repos) cqrs.Handler[*in.PinChatMessageRequest, *out.ChatConversationResponse] {
	return &pinChatMessageHandler{baseRepo: baseRepo}
}

func (h *pinChatMessageHandler) Handle(ctx context.Context, req *in.PinChatMessageRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := h.baseRepo.RoomAggregateRepository().Load(ctx, req.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := agg.PinMessage(accountID, req.MessageID, time.Now().UTC(), accountID); err != nil {
		return nil, stackErr.Error(err)
	}
	lastMessage := lastPendingMessage(agg.PendingMessages())
	if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
		return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, agg))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	res, err := roomsupport.BuildConversationResultFromState(ctx, h.baseRepo, accountID, agg.Room(), agg.Members(), lastMessage, true)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return roomsupport.ToConversationResponse(res), nil
}
