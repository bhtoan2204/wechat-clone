package command

import (
	"context"
	"time"

	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	"wechat-clone/core/modules/room/application/service"
	roomsupport "wechat-clone/core/modules/room/application/support"
	roomrepos "wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type pinChatMessageHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewPinChatMessageHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.PinChatMessageRequest, *out.ChatRoomCommandResponse] {
	return &pinChatMessageHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *pinChatMessageHandler) Handle(ctx context.Context, req *in.PinChatMessageRequest) (*out.ChatRoomCommandResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := h.baseRepo.RoomAggregateRepository().Load(ctx, req.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := agg.PinMessage(accountID, req.MessageID, time.Now().UTC(), accountID); err != nil {
		return nil, stackErr.Error(err)
	}
	if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
		return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, agg))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.ChatRoomCommandResponse{RoomID: agg.Room().ID, Status: CommandStatusUpdated}, nil
}
