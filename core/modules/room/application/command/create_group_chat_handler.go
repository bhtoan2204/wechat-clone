package command

import (
	"context"
	"fmt"
	"time"

	"go-socket/core/modules/room/application/dto/in"
	"go-socket/core/modules/room/application/dto/out"
	roomsupport "go-socket/core/modules/room/application/support"
	"go-socket/core/modules/room/domain/aggregate"
	"go-socket/core/modules/room/domain/entity"
	roomrepos "go-socket/core/modules/room/domain/repos"
	roomtypes "go-socket/core/modules/room/types"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

type createGroupChatHandler struct {
	baseRepo roomrepos.Repos
}

func NewCreateGroupChatHandler(baseRepo roomrepos.Repos) cqrs.Handler[*in.CreateGroupChatRequest, *out.ChatConversationResponse] {
	return &createGroupChatHandler{baseRepo: baseRepo}
}

func (h *createGroupChatHandler) Handle(ctx context.Context, req *in.CreateGroupChatRequest) (*out.ChatConversationResponse, error) {
	accountID, err := roomsupport.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := ensureProjectedAccountsExist(ctx, h.baseRepo, req.MemberIDs...); err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	room, err := entity.NewRoom(uuid.NewString(), req.Name, req.Description, accountID, roomtypes.RoomTypeGroup, "", now)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	memberSet, err := entity.BuildGroupMemberRoles(accountID, req.MemberIDs)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	members := make([]*entity.RoomMemberEntity, 0, len(memberSet))
	for memberID, role := range memberSet {
		member, createErr := entity.NewRoomMember(uuid.NewString(), room.ID, memberID, role, now)
		if createErr != nil {
			return nil, stackErr.Error(createErr)
		}
		members = append(members, member)
	}

	agg, err := aggregate.NewConversationRoomAggregate(
		room,
		members,
		accountID,
		fmt.Sprintf("%s created the group", accountID),
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
