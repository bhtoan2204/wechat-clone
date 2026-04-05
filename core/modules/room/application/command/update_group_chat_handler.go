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

type updateGroupChatHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewUpdateGroupChatHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.UpdateGroupChatRequest, *out.ChatConversationResponse] {
	return &updateGroupChatHandler{roomService: roomService}
}
func (h *updateGroupChatHandler) Handle(ctx context.Context, req *in.UpdateGroupChatRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	res, err := h.roomService.UpdateGroup(ctx, accountID, req.RoomID, apptypes.UpdateGroupCommand{
		Name:        req.Name,
		Description: req.Description,
	})
	if err != nil {
		return nil, err
	}
	return roomsupport.ToConversationResponse(res), nil
}
