package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	"wechat-clone/core/modules/room/application/dto/in"
	"wechat-clone/core/modules/room/application/dto/out"
	roomsupport "wechat-clone/core/modules/room/application/support"
	"wechat-clone/core/modules/room/domain/aggregate"
	"wechat-clone/core/modules/room/domain/entity"
	roomrepos "wechat-clone/core/modules/room/domain/repos"
	roomtypes "wechat-clone/core/modules/room/types"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type createDirectConversationHandler struct {
	baseRepo roomrepos.Repos
}

func NewCreateDirectConversationHandler(baseRepo roomrepos.Repos) cqrs.Handler[*in.CreateDirectConversationRequest, *out.ChatConversationResponse] {
	return &createDirectConversationHandler{baseRepo: baseRepo}
}

func (h *createDirectConversationHandler) Handle(ctx context.Context, req *in.CreateDirectConversationRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	// if err := ensureProjectedAccountsExist(ctx, h.baseRepo, accountID, req.PeerAccountID); err != nil {
	// 	return nil, stackErr.Error(err)
	// }

	now := time.Now().UTC()
	room, err := entity.NewDirectConversationRoom(uuid.NewString(), accountID, req.PeerAccountID, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	existing, err := h.baseRepo.RoomAggregateRepository().LoadByDirectKey(ctx, room.DirectKey)
	if err == nil && existing != nil {
		res, buildErr := roomsupport.BuildConversationResultFromState(ctx, h.baseRepo, accountID, existing.Room(), existing.Members(), nil, true)
		if buildErr != nil {
			return nil, stackErr.Error(buildErr)
		}
		return roomsupport.ToConversationResponse(res), nil
	}
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, stackErr.Error(err)
	}

	ownerMember, err := entity.NewRoomMember(uuid.NewString(), room.ID, accountID, roomtypes.RoomRoleOwner, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	peerMember, err := entity.NewRoomMember(uuid.NewString(), room.ID, req.PeerAccountID, roomtypes.RoomRoleMember, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	agg, err := aggregate.NewConversationRoomAggregate(
		room,
		[]*entity.RoomMemberEntity{ownerMember, peerMember},
		accountID,
		fmt.Sprintf("%s started a direct conversation", ownerMember.DisplayName),
		now,
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	lastMessage := lastPendingMessage(agg.PendingMessages())

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
		return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, agg))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	res, err := roomsupport.BuildConversationResultFromState(ctx, h.baseRepo, accountID, room, agg.Members(), lastMessage, true)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return roomsupport.ToConversationResponse(res), nil
}
