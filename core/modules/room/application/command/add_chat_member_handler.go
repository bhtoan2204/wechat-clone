package command

import (
	"context"
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
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type addChatMemberHandler struct {
	baseRepo roomrepos.Repos
	realtime service.RealtimeService
}

func NewAddChatMemberHandler(baseRepo roomrepos.Repos, realtime service.RealtimeService) cqrs.Handler[*in.AddChatMemberRequest, *out.ChatConversationResponse] {
	return &addChatMemberHandler{baseRepo: baseRepo, realtime: realtime}
}

func (h *addChatMemberHandler) Handle(ctx context.Context, req *in.AddChatMemberRequest) (*out.ChatConversationResponse, error) {
	log := logging.FromContext(ctx)
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := ensureProjectedAccountsExist(ctx, h.baseRepo, req.AccountID); err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := h.baseRepo.RoomAggregateRepository().Load(ctx, req.RoomID)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	member, err := entity.NewRoomMember(uuid.NewString(), req.RoomID, req.AccountID, types.RoomRole(req.Role), now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	added, err := agg.AddMember(accountID, member, now, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	lastMessage := lastPendingMessage(agg.PendingMessages())
	if added {
		if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
			return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, agg))
		}); err != nil {
			return nil, stackErr.Error(err)
		}
	}

	res, err := roomsupport.BuildConversationResultFromState(ctx, h.baseRepo, accountID, agg.Room(), agg.Members(), lastMessage, true)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	out := roomsupport.ToConversationResponse(res)
	if err := h.realtime.EmitMessage(ctx, types.MessagePayload{
		RoomId:  out.RoomID,
		Type:    reflect.TypeOf(out).Elem().Name(),
		Payload: out,
	}); err != nil {
		log.Warnw("Emit msg failed", zap.Error(err))
	}

	return out, nil
}
