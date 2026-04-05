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

type deleteChatMessageHandler struct {
	messageService *roomservice.MessageCommandService
}

func NewDeleteChatMessageHandler(messageService *roomservice.MessageCommandService) cqrs.Handler[*in.DeleteChatMessageRequest, *out.DeleteChatMessageResponse] {
	return &deleteChatMessageHandler{messageService: messageService}
}

func (h *deleteChatMessageHandler) Handle(ctx context.Context, req *in.DeleteChatMessageRequest) (*out.DeleteChatMessageResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	if err := h.messageService.DeleteMessage(ctx, accountID, req.MessageID, apptypes.DeleteMessageCommand{
		Scope: req.Scope,
	}); err != nil {
		return nil, err
	}

	return &out.DeleteChatMessageResponse{OK: true}, nil
}
