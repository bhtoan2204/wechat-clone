package command

import (
	"context"
	"time"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomsupport "go-socket/core/modules/room/application/support"
	"go-socket/core/modules/room/domain/aggregate"
	"go-socket/core/modules/room/domain/entity"
	roomrepos "go-socket/core/modules/room/domain/repos"
	roomtypes "go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

type createRoomHandler struct {
	baseRepo roomrepos.Repos
}

func NewCreateRoomHandler(baseRepo roomrepos.Repos) cqrs.Handler[*in.CreateRoomRequest, *out.CreateRoomResponse] {
	return &createRoomHandler{
		baseRepo: baseRepo,
	}
}

func (h *createRoomHandler) Handle(ctx context.Context, req *in.CreateRoomRequest) (*out.CreateRoomResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	room, err := entity.NewRoom(uuid.NewString(), req.Name, req.Description, accountID, roomtypes.RoomType(req.RoomType), "", now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := aggregate.NewRoomStateAggregate(room, 0)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	agg.RecordCreated(1, now)

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
		return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, agg))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.CreateRoomResponse{
		ID:   room.ID,
		Name: room.Name,
	}, nil
}
