package command

import (
	"context"
	"fmt"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type deleteRoomHandler struct {
	roomRepo repos.RoomRepository
}

func NewDeleteRoomHandler(roomRepo repos.RoomRepository) cqrs.Handler[*in.DeleteRoomRequest, *out.DeleteRoomResponse] {
	return &deleteRoomHandler{
		roomRepo: roomRepo,
	}
}

func (h *deleteRoomHandler) Handle(ctx context.Context, req *in.DeleteRoomRequest) (*out.DeleteRoomResponse, error) {
	log := logging.FromContext(ctx).Named("DeleteRoom")
	err := h.roomRepo.DeleteRoom(ctx, req.Id)
	if err != nil {
		log.Errorw("Failed to delete room", zap.Error(err))
		return nil, stackerr.Error(fmt.Errorf("delete room failed: %w", err))
	}
	return &out.DeleteRoomResponse{
		Message: "Room deleted successfully",
	}, nil
}
