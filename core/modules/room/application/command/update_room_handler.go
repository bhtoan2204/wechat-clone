package command

import (
	"context"
	"time"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomrepos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"
)

type updateRoomHandler struct {
	baseRepo roomrepos.Repos
}

func NewUpdateRoomHandler(baseRepo roomrepos.Repos) cqrs.Handler[*in.UpdateRoomRequest, *out.UpdateRoomResponse] {
	return &updateRoomHandler{
		baseRepo: baseRepo,
	}
}

func (h *updateRoomHandler) Handle(ctx context.Context, req *in.UpdateRoomRequest) (*out.UpdateRoomResponse, error) {
	agg, err := h.baseRepo.RoomAggregateRepository().Load(ctx, req.ID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	updated, err := agg.UpdateRoomDetails(req.Name, "", "", time.Now().UTC())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if updated {
		if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
			return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, agg))
		}); err != nil {
			return nil, stackErr.Error(err)
		}
	}

	return &out.UpdateRoomResponse{
		ID:        agg.Room().ID,
		Name:      agg.Room().Name,
		CreatedAt: agg.Room().CreatedAt.UTC().Format(time.RFC3339),
		UpdatedAt: agg.Room().UpdatedAt.UTC().Format(time.RFC3339),
	}, nil
}
