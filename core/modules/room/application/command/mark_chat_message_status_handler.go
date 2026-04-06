package command

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

type markChatMessageStatusHandler struct {
	messageService *roomservice.MessageCommandService
}

func NewMarkChatMessageStatusHandler(messageService *roomservice.MessageCommandService) cqrs.Handler[*in.MarkChatMessageStatusRequest, *out.MarkChatMessageStatusResponse] {
	return &markChatMessageStatusHandler{messageService: messageService}
}

func (h *markChatMessageStatusHandler) Handle(ctx context.Context, req *in.MarkChatMessageStatusRequest) (*out.MarkChatMessageStatusResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := h.messageService.MarkMessageStatus(ctx, accountID, req.MessageID, apptypes.MarkMessageStatusCommand{
		Status: req.Status,
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.MarkChatMessageStatusResponse{Ok: true}, nil
}
