package command

import (
	"context"
	"fmt"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	stackerr "go-socket/core/shared/pkg/stackErr"
)

type joinRoomHandler struct {
	roomRepo repos.RoomRepository
}

func NewJoinRoomHandler(roomRepo repos.RoomRepository) cqrs.Handler[*in.JoinRoomRequest, *out.JoinRoomResponse] {
	return &joinRoomHandler{
		roomRepo: roomRepo,
	}
}

func (h *joinRoomHandler) Handle(ctx context.Context, req *in.JoinRoomRequest) (*out.JoinRoomResponse, error) {
	return nil, stackerr.Error(fmt.Errorf("not implemented"))
}
