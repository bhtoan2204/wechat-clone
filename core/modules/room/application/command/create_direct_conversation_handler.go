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

type createDirectConversationHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewCreateDirectConversationHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.CreateDirectConversationRequest, *out.ChatConversationResponse] {
	return &createDirectConversationHandler{roomService: roomService}
}

func (h *createDirectConversationHandler) Handle(ctx context.Context, req *in.CreateDirectConversationRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	res, err := h.roomService.CreateDirectConversation(ctx, accountID, apptypes.CreateDirectConversationCommand{
		PeerAccountID: req.PeerAccountID,
	})
	if err != nil {
		return nil, err
	}
	return roomsupport.ToConversationResponse(res), nil
}
