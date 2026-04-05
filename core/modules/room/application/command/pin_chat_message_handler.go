package command

import (
	"context"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
)

type pinChatMessageHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewPinChatMessageHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.PinChatMessageRequest, *out.ChatConversationResponse] {
	return &pinChatMessageHandler{roomService: roomService}
}

func (h *pinChatMessageHandler) Handle(ctx context.Context, req *in.PinChatMessageRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	res, err := h.roomService.PinMessage(ctx, accountID, req.RoomID, apptypes.PinMessageCommand{
		MessageID: req.MessageID,
	})
	if err != nil {
		return nil, err
	}

	return roomsupport.ToConversationResponse(res), nil
}
