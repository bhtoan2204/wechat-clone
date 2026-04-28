package command

import (
	"context"
	"time"

	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	"wechat-clone/core/modules/room/application/service"
	roomsupport "wechat-clone/core/modules/room/application/support"
	"wechat-clone/core/modules/room/domain/entity"
	roomrepos "wechat-clone/core/modules/room/domain/repos"
	roomtypes "wechat-clone/core/modules/room/types"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

type addChatMemberHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewAddChatMemberHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.AddChatMemberRequest, *out.ChatRoomCommandResponse] {
	return &addChatMemberHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *addChatMemberHandler) Handle(ctx context.Context, req *in.AddChatMemberRequest) (*out.ChatRoomCommandResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	agg, err := h.baseRepo.RoomAggregateRepository().Load(ctx, req.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	member, err := entity.NewRoomMember(uuid.NewString(), req.RoomID, req.AccountID, roomtypes.RoomRole(req.Role), now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	added, err := agg.AddMember(accountID, member, now, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if added {
		if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
			return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, agg))
		}); err != nil {
			return nil, stackErr.Error(err)
		}
	}

	return &out.ChatRoomCommandResponse{RoomID: agg.Room().ID, Status: commandStatus(added)}, nil
}
