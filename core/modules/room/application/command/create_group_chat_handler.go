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

type createGroupChatHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewCreateGroupChatHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.CreateGroupChatRequest, *out.ChatConversationResponse] {
	return &createGroupChatHandler{roomService: roomService}
}

func (h *createGroupChatHandler) Handle(ctx context.Context, req *in.CreateGroupChatRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	res, err := h.roomService.CreateGroup(ctx, accountID, apptypes.CreateGroupCommand{
		Name:        req.Name,
		Description: req.Description,
		MemberIDs:   req.MemberIDs,
	})
	if err != nil {
		return nil, err
	}
	return roomsupport.ToConversationResponse(res), nil
}
