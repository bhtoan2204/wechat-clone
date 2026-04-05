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

type removeChatMemberHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewRemoveChatMemberHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.RemoveChatMemberRequest, *out.ChatConversationResponse] {
	return &removeChatMemberHandler{roomService: roomService}
}
func (h *removeChatMemberHandler) Handle(ctx context.Context, req *in.RemoveChatMemberRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	res, err := h.roomService.RemoveMember(ctx, accountID, req.RoomID, apptypes.RemoveMemberCommand{
		AccountID: req.AccountID,
	})
	if err != nil {
		return nil, err
	}
	return roomsupport.ToConversationResponse(res), nil
}
