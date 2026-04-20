package command

import (
	"context"
	"reflect"
	"time"

	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	"wechat-clone/core/modules/room/application/service"
	roomsupport "wechat-clone/core/modules/room/application/support"
	roomrepos "wechat-clone/core/modules/room/domain/repos"
	"wechat-clone/core/modules/room/types"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type toggleChatMessageReactionHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewToggleChatMessageReactionHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.ToggleChatMessageReactionRequest, *out.ChatMessageResponse] {
	return &toggleChatMessageReactionHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *toggleChatMessageReactionHandler) Handle(ctx context.Context, req *in.ToggleChatMessageReactionRequest) (*out.ChatMessageResponse, error) {
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

	res, err := roomsupport.BuildMessageResultFromState(ctx, h.baseRepo, accountID, agg.Message())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	out := roomsupport.ToMessageResponse(res)
	if err := h.realtime.EmitMessage(ctx, types.MessagePayload{
		RoomId:  out.RoomID,
		Type:    reflect.TypeOf(out).Elem().Name(),
		Payload: out,
	}); err != nil {
		logging.FromContext(ctx).Warnw("failed to emit realtime message after toggling chat message reaction", zap.Error(err), "message_id", req.MessageID)
	}
	return out, nil
}
