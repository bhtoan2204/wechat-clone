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

type editChatMessageHandler struct {
	messageService *roomservice.MessageCommandService
}

func NewEditChatMessageHandler(messageService *roomservice.MessageCommandService) cqrs.Handler[*in.EditChatMessageRequest, *out.ChatMessageResponse] {
	return &editChatMessageHandler{messageService: messageService}
}

func (h *editChatMessageHandler) Handle(ctx context.Context, req *in.EditChatMessageRequest) (*out.ChatMessageResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}

	res, err := h.messageService.EditMessage(ctx, accountID, req.MessageID, apptypes.EditMessageCommand{
		Message: req.Message,
	})
	if err != nil {
		return nil, err
	}

	return roomsupport.ToMessageResponse(res), nil
}
