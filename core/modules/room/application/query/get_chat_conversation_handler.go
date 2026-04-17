package query

import (
	"context"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type getChatConversationHandler struct {
	conversations roomservice.ConversationQueryService
}

func NewGetChatConversationHandler(conversations roomservice.ConversationQueryService) cqrs.Handler[*in.GetChatConversationRequest, *out.ChatConversationResponse] {
	return &getChatConversationHandler{conversations: conversations}
}

func (h *getChatConversationHandler) Handle(ctx context.Context, req *in.GetChatConversationRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	res, err := h.conversations.GetConversation(ctx, accountID, apptypes.GetConversationQuery{
		RoomID: req.RoomID,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return roomsupport.ToConversationResponse(res), nil
}
