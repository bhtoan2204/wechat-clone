package query

import (
	"context"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
)

type getChatConversationHandler struct {
	chatService *roomservice.ChatQueryService
}

func NewGetChatConversationHandler(chatService *roomservice.ChatQueryService) cqrs.Handler[*in.GetChatConversationRequest, *out.ChatConversationResponse] {
	return &getChatConversationHandler{chatService: chatService}
}

func (h *getChatConversationHandler) Handle(ctx context.Context, req *in.GetChatConversationRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	res, err := h.chatService.GetConversation(ctx, accountID, apptypes.GetConversationQuery{
		RoomID: req.RoomID,
	})
	if err != nil {
		return nil, err
	}

	return roomsupport.ToConversationResponse(res), nil
}
