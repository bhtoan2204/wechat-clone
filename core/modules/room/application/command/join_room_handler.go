package command

import (
	"context"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
	stackerr "go-socket/core/shared/pkg/stackErr"
)

type joinRoomHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewJoinRoomHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.JoinRoomRequest, *out.JoinRoomResponse] {
	return &joinRoomHandler{
		roomService: roomService,
	}
}

func (h *joinRoomHandler) Handle(ctx context.Context, req *in.JoinRoomRequest) (*out.JoinRoomResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackerr.Error(err)
	}
	if err := h.roomService.JoinRoom(ctx, accountID, apptypes.JoinRoomCommand{RoomID: req.RoomID}); err != nil {
		return nil, stackerr.Error(err)
	}
	return &out.JoinRoomResponse{Message: "join room is scaffolded"}, nil
}
