package command

import (
	"context"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomtypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type updateRoomHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewUpdateRoomHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.UpdateRoomRequest, *out.UpdateRoomResponse] {
	return &updateRoomHandler{
		roomService: roomService,
	}
}

func (h *updateRoomHandler) Handle(ctx context.Context, req *in.UpdateRoomRequest) (*out.UpdateRoomResponse, error) {
	log := logging.FromContext(ctx).Named("UpdateRoom")
	room, err := h.roomService.UpdateRoom(ctx, "", req.ID, roomtypes.UpdateRoomCommand{
		Name: req.Name,
	})
	if err != nil {
		log.Errorw("Failed to get room", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	return &out.UpdateRoomResponse{
		ID:        room.ID,
		Name:      room.Name,
		CreatedAt: room.CreatedAt,
		UpdatedAt: room.UpdatedAt,
	}, nil
}
