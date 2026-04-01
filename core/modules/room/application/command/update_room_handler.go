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
	"time"

	"go.uber.org/zap"
)

type updateRoomHandler struct {
	roomRepo repos.RoomRepository
}

func NewUpdateRoomHandler(roomRepo repos.RoomRepository) cqrs.Handler[*in.UpdateRoomRequest, *out.UpdateRoomResponse] {
	return &updateRoomHandler{
		roomRepo: roomRepo,
	}
}

func (h *updateRoomHandler) Handle(ctx context.Context, req *in.UpdateRoomRequest) (*out.UpdateRoomResponse, error) {
	log := logging.FromContext(ctx).Named("UpdateRoom")
	room, err := h.roomRepo.GetRoomByID(ctx, req.Id)
	if err != nil {
		log.Errorw("Failed to get room", zap.Error(err))
		return nil, stackerr.Error(fmt.Errorf("get room failed: %w", err))
	}
	room.Name = req.Name
	err = h.roomRepo.UpdateRoom(ctx, room)
	if err != nil {
		log.Errorw("Failed to update room", zap.Error(err))
		return nil, stackerr.Error(fmt.Errorf("update room failed: %w", err))
	}
	return &out.UpdateRoomResponse{
		Id:        room.ID,
		Name:      room.Name,
		CreatedAt: room.CreatedAt.Format(time.RFC3339),
		UpdatedAt: room.UpdatedAt.Format(time.RFC3339),
	}, nil
}
