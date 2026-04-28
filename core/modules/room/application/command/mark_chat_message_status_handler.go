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

type markChatMessageStatusHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewMarkChatMessageStatusHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.MarkChatMessageStatusRequest, *out.ChatMessageCommandResponse] {
	return &markChatMessageStatusHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *markChatMessageStatusHandler) Handle(ctx context.Context, req *in.MarkChatMessageStatusRequest) (*out.ChatMessageCommandResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := h.baseRepo.MessageAggregateRepository().LoadForRecipient(ctx, req.MessageID, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	changed, err := agg.MarkStatus(accountID, req.Status, nil, time.Now().UTC())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if changed {
		if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
			return stackErr.Error(txRepos.MessageAggregateRepository().Save(ctx, agg))
		}); err != nil {
			return nil, stackErr.Error(err)
		}
	}

	return &out.ChatMessageCommandResponse{MessageID: agg.Message().ID, RoomID: agg.Message().RoomID, Status: commandStatus(changed)}, nil
}
