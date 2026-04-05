package query

import (
	"context"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type getRoomHandler struct {
	roomQueryService *roomservice.RoomQueryService
}

func NewGetRoomHandler(roomQueryService *roomservice.RoomQueryService) cqrs.Handler[*in.GetRoomRequest, *out.GetRoomResponse] {
	return &getRoomHandler{
		roomQueryService: roomQueryService,
	}
}

func (h *getRoomHandler) Handle(ctx context.Context, req *in.GetRoomRequest) (*out.GetRoomResponse, error) {
	log := logging.FromContext(ctx).Named("GetRoom")
	room, err := h.roomQueryService.GetRoom(ctx, apptypes.GetRoomQuery{ID: req.Id})
	if err != nil {
		log.Errorw("Failed to get room", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	return roomsupport.ToGetRoomResponse(room), nil
}
