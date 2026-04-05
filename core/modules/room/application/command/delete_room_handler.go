package command

import (
	"context"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type deleteRoomHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewDeleteRoomHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.DeleteRoomRequest, *out.DeleteRoomResponse] {
	return &deleteRoomHandler{
		roomService: roomService,
	}
}

func (h *deleteRoomHandler) Handle(ctx context.Context, req *in.DeleteRoomRequest) (*out.DeleteRoomResponse, error) {
	log := logging.FromContext(ctx).Named("DeleteRoom")
	if err := h.roomService.DeleteRoom(ctx, req.Id); err != nil {
		log.Errorw("Failed to delete room", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	return &out.DeleteRoomResponse{
		Message: "Room deleted successfully",
	}, nil
}
