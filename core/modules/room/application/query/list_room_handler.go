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

type listRoomHandler struct {
	roomQueryService *roomservice.RoomQueryService
}

func NewListRoomHandler(roomQueryService *roomservice.RoomQueryService) cqrs.Handler[*in.ListRoomsRequest, *out.ListRoomsResponse] {
	return &listRoomHandler{
		roomQueryService: roomQueryService,
	}
}

func (h *listRoomHandler) Handle(ctx context.Context, req *in.ListRoomsRequest) (*out.ListRoomsResponse, error) {
	log := logging.FromContext(ctx).Named("ListRooms")
	rooms, err := h.roomQueryService.ListRooms(ctx, apptypes.ListRoomsQuery{Page: req.Page, Limit: req.Limit})
	if err != nil {
		log.Errorw("Failed to list rooms", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	return roomsupport.ToListRoomsResponse(rooms), nil
}
