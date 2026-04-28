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

type toggleChatMessageReactionHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewToggleChatMessageReactionHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.ToggleChatMessageReactionRequest, *out.ChatMessageCommandResponse] {
	return &toggleChatMessageReactionHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *toggleChatMessageReactionHandler) Handle(ctx context.Context, req *in.ToggleChatMessageReactionRequest) (*out.ChatMessageCommandResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := h.baseRepo.MessageAggregateRepository().Load(ctx, req.MessageID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := agg.ToggleReaction(accountID, req.Emoji, time.Now().UTC()); err != nil {
		return nil, stackErr.Error(err)
	}
	if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
		return stackErr.Error(txRepos.MessageAggregateRepository().Save(ctx, agg))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.ChatMessageCommandResponse{MessageID: agg.Message().ID, RoomID: agg.Message().RoomID, Status: CommandStatusUpdated}, nil
}
