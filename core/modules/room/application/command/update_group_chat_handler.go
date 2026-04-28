package command

import (
	"context"
	"time"

	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	"wechat-clone/core/modules/room/application/service"
	roomsupport "wechat-clone/core/modules/room/application/support"
	"wechat-clone/core/modules/room/domain/aggregate"
	roomrepos "wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type updateGroupChatHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewUpdateGroupChatHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.UpdateGroupChatRequest, *out.ChatRoomCommandResponse] {
	return &updateGroupChatHandler{baseRepo: baseRepo, realtime: realtime}
}
func (h *updateGroupChatHandler) Handle(ctx context.Context, req *in.UpdateGroupChatRequest) (*out.ChatRoomCommandResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := h.baseRepo.RoomAggregateRepository().Load(ctx, req.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	updated, err := agg.UpdateGroupDetails(aggregate.UpdateGroupDetailsParams{
		ActorID:       accountID,
		Name:          req.Name,
		Description:   req.Description,
		Now:           time.Now().UTC(),
		SystemActorID: accountID,
	})
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

	return &out.ChatRoomCommandResponse{RoomID: agg.Room().ID, Status: commandStatus(updated)}, nil
}
