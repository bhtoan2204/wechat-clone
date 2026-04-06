package command

import (
	"context"
	"errors"
	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomservice "go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	apptypes "go-socket/core/modules/room/application/types"
	roomtypes "go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type createRoomHandler struct {
	roomService *roomservice.RoomCommandService
}

func NewCreateRoomHandler(roomService *roomservice.RoomCommandService) cqrs.Handler[*in.CreateRoomRequest, *out.CreateRoomResponse] {
	return &createRoomHandler{
		roomService: roomService,
	}
}

func (h *createRoomHandler) Handle(ctx context.Context, req *in.CreateRoomRequest) (*out.CreateRoomResponse, error) {
	log := logging.FromContext(ctx).Named("CreateRoom")
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		log.Errorw("Account not found", zap.Error(err))
		return nil, stackErr.Error(errors.New("account not found"))
	}
	room, err := h.roomService.CreateRoom(ctx, accountID, apptypes.CreateRoomCommand{
		Name:        req.Name,
		Description: req.Description,
		RoomType:    roomtypes.RoomType(req.RoomType),
	})
	if err != nil {
		log.Errorw("Failed to create room", zap.Error(err), zap.Any("room", room))
		return nil, stackErr.Error(err)
	}

	return &out.CreateRoomResponse{
		ID:   room.ID,
		Name: room.Name,
	}, nil
}
