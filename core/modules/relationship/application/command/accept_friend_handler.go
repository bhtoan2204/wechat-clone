package command

import (
	"context"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/relationship/application/dto/in"
	"wechat-clone/core/modules/relationship/application/dto/out"
	repos "wechat-clone/core/modules/relationship/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

type acceptFriendRequestHandler struct {
	baseRepo repos.Repos
}

func NewAcceptFriendRequest(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.AcceptFriendRequestRequest, *out.AcceptFriendRequestResponse] {
	return &acceptFriendRequestHandler{
		baseRepo: baseRepo,
	}
}

func (u *acceptFriendRequestHandler) Handle(ctx context.Context, req *in.AcceptFriendRequestRequest) (*out.AcceptFriendRequestResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	response := &out.AcceptFriendRequestResponse{}
	now := nowUTC()

	if err := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		pairAgg, err := txRepos.RelationshipPairAggregateRepository().LoadForUpdate(ctx, accountID, req.RequesterUserID)
		if err != nil {
			return stackErr.Error(err)
		}

		if err := pairAgg.AcceptFriendRequest(uuid.NewString(), now); err != nil {
			return stackErr.Error(err)
		}

		friendship := pairAgg.FriendshipCreated()
		friendID, err := friendship.OtherUserID(accountID)
		if err != nil {
			return stackErr.Error(err)
		}

		if err := txRepos.RelationshipPairAggregateRepository().Save(ctx, pairAgg); err != nil {
			return stackErr.Error(err)
		}

		response.FriendshipID = friendship.ID
		response.UserID = accountID
		response.FriendID = friendID
		response.CreatedAt = friendship.CreatedAt.Unix()
		return nil
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return response, nil
}
