package query

import (
	"context"
	"fmt"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/domain/entity"
	"go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"
	"time"

	"github.com/samber/lo"
	"go.uber.org/zap"
)

type listRoomHandler struct {
	roomRepo repos.RoomRepository
}

func NewListRoomHandler(roomRepo repos.RoomRepository) cqrs.Handler[*in.ListRoomsRequest, *out.ListRoomsResponse] {
	return &listRoomHandler{
		roomRepo: roomRepo,
	}
}

func (h *listRoomHandler) Handle(ctx context.Context, req *in.ListRoomsRequest) (*out.ListRoomsResponse, error) {
	log := logging.FromContext(ctx).Named("ListRooms")
	rooms, err := h.roomRepo.ListRooms(ctx, utils.QueryOptions{
		Offset: &req.Page,
		Limit:  &req.Limit,
	})
	if err != nil {
		log.Errorw("Failed to list rooms", zap.Error(err))
		return nil, stackerr.Error(fmt.Errorf("list rooms failed: %w", err))
	}
	return &out.ListRoomsResponse{
		Rooms: lo.Map(rooms, func(room *entity.Room, _ int) out.RoomResponse {
			return out.RoomResponse{
				Id:          room.ID,
				Name:        room.Name,
				Description: room.Description,
				RoomType:    string(room.RoomType),
				OwnerId:     room.OwnerID,
				CreatedAt:   room.CreatedAt.Format(time.RFC3339),
				UpdatedAt:   room.UpdatedAt.Format(time.RFC3339),
			}
		}),
	}, nil
}
