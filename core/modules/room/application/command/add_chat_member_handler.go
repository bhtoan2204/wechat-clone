package command

import (
	"context"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/cqrs"
)

type addChatMemberHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewAddChatMemberHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.AddChatMemberRequest, *out.ChatConversationResponse] {
	return &addChatMemberHandler{roomService: roomService}
}
func (h *addChatMemberHandler) Handle(ctx context.Context, req *in.AddChatMemberRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, err
	}
	res, err := h.roomService.AddMember(ctx, accountID, req.RoomID, apptypes.AddMemberCommand{
		AccountID: req.AccountID,
		Role:      types.RoomRole(req.Role),
	})
	if err != nil {
		return nil, err
	}
	return roomsupport.ToConversationResponse(res), nil
}
