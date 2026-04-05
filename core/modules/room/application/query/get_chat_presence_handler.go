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

type getChatPresenceHandler struct {
	chatService *roomservice.ChatQueryService
}

func NewGetChatPresenceHandler(chatService *roomservice.ChatQueryService) cqrs.Handler[*in.GetChatPresenceRequest, *out.ChatPresenceResponse] {
	return &getChatPresenceHandler{chatService: chatService}
}

func (h *getChatPresenceHandler) Handle(ctx context.Context, req *in.GetChatPresenceRequest) (*out.ChatPresenceResponse, error) {
	res, err := h.chatService.GetPresence(ctx, apptypes.GetPresenceQuery{AccountID: req.AccountID})
	if err != nil {
		return nil, err
	}

	return roomsupport.ToPresenceResponse(res), nil
}
