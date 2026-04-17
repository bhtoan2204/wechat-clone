package command

import (
	"context"
	"errors"
	"reflect"
	"time"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	"go-socket/core/modules/room/application/service"
	roomsupport "go-socket/core/modules/room/application/support"
	"go-socket/core/modules/room/domain/entity"
	roomrepos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type markChatMessageStatusHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewMarkChatMessageStatusHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.MarkChatMessageStatusRequest, *out.MarkChatMessageStatusResponse] {
	return &markChatMessageStatusHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *markChatMessageStatusHandler) Handle(ctx context.Context, req *in.MarkChatMessageStatusRequest) (*out.MarkChatMessageStatusResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := h.baseRepo.MessageAggregateRepository().Load(ctx, req.MessageID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var member = (*entity.RoomMemberEntity)(nil)
	if roomMember, memberErr := h.baseRepo.RoomMemberRepository().GetRoomMemberByAccount(ctx, agg.Message().RoomID, accountID); memberErr == nil {
		member = roomMember
	} else if !errors.Is(memberErr, gorm.ErrRecordNotFound) {
		return nil, stackErr.Error(memberErr)
	}

	changed, err := agg.MarkStatus(accountID, req.Status, member, time.Now().UTC())
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

	out := &out.MarkChatMessageStatusResponse{Ok: true}
	h.realtime.EmitMessage(ctx, types.MessagePayload{
		RoomId:  agg.Message().RoomID,
		Type:    reflect.TypeOf(out).Elem().Name(),
		Payload: out,
	})
	return out, nil
}
