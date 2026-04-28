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

func NewCreateDirectConversationHandler(baseRepo roomrepos.Repos) cqrs.Handler[*in.CreateDirectConversationRequest, *out.ChatRoomCommandResponse] {
	return &createDirectConversationHandler{baseRepo: baseRepo}
}

func (h *createDirectConversationHandler) Handle(ctx context.Context, req *in.CreateDirectConversationRequest) (*out.ChatRoomCommandResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	now := time.Now().UTC()
	room, err := entity.NewDirectConversationRoom(uuid.NewString(), accountID, req.PeerAccountID, now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	existing, err := h.baseRepo.RoomAggregateRepository().LoadByDirectKey(ctx, room.DirectKey)
	if err == nil && existing != nil {
		return &out.ChatRoomCommandResponse{RoomID: existing.Room().ID, Status: CommandStatusAlreadyExists}, nil
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

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos roomrepos.Repos) error {
		return stackErr.Error(txRepos.RoomAggregateRepository().Save(ctx, agg))
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.ChatRoomCommandResponse{RoomID: room.ID, Status: CommandStatusCreated}, nil
}
