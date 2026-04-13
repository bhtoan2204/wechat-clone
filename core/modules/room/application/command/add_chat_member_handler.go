package command

import (
	"context"
	"time"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomsupport "go-socket/core/modules/room/application/support"
	"go-socket/core/modules/room/domain/entity"
	roomrepos "go-socket/core/modules/room/domain/repos"
	"go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

type addChatMemberHandler struct {
	baseRepo roomrepos.Repos
}

func NewAddChatMemberHandler(baseRepo roomrepos.Repos) cqrs.Handler[*in.AddChatMemberRequest, *out.ChatConversationResponse] {
	return &addChatMemberHandler{baseRepo: baseRepo}
}
func (h *addChatMemberHandler) Handle(ctx context.Context, req *in.AddChatMemberRequest) (*out.ChatConversationResponse, error) {
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
	return roomsupport.ToConversationResponse(res), nil
}
