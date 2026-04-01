package query

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

type getRoomHandler struct {
	roomRepo repos.RoomRepository
}

func NewGetRoomHandler(roomRepo repos.RoomRepository) cqrs.Handler[*in.GetRoomRequest, *out.GetRoomResponse] {
	return &getRoomHandler{
		roomRepo: roomRepo,
	}
}

func (h *getRoomHandler) Handle(ctx context.Context, req *in.GetRoomRequest) (*out.GetRoomResponse, error) {
	log := logging.FromContext(ctx).Named("GetRoom")
	room, err := h.roomRepo.GetRoomByID(ctx, req.Id)
	if err != nil {
		log.Errorw("Failed to get room", zap.Error(err))
		return nil, stackerr.Error(fmt.Errorf("get room failed: %w", err))
	}
	return &out.GetRoomResponse{
		Id:          room.ID,
		Name:        room.Name,
		Description: room.Description,
		RoomType:    string(room.RoomType),
		OwnerId:     room.OwnerID,
		CreatedAt:   room.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   room.UpdatedAt.Format(time.RFC3339),
	}, nil
}
