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

type forwardChatMessageHandler struct {
	messageService *roomservice.MessageCommandService
}

func NewForwardChatMessageHandler(messageService *roomservice.MessageCommandService) cqrs.Handler[*in.ForwardChatMessageRequest, *out.ChatMessageResponse] {
	return &forwardChatMessageHandler{messageService: messageService}
}

func (h *forwardChatMessageHandler) Handle(ctx context.Context, req *in.ForwardChatMessageRequest) (*out.ChatMessageResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	res, err := h.messageService.ForwardMessage(ctx, accountID, req.MessageID, apptypes.ForwardMessageCommand{
		TargetRoomID: req.TargetRoomID,
	})
	if err != nil {
		return nil, err
	}

	return roomsupport.ToMessageResponse(res), nil
}
