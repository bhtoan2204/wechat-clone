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

type sendFriendRequestHandler struct {
	baseRepo repos.Repos
}

func NewSendFriendRequest(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.SendFriendRequestRequest, *out.SendFriendRequestResponse] {
	return &sendFriendRequestHandler{
		baseRepo: baseRepo,
	}
}

func (u *sendFriendRequestHandler) Handle(ctx context.Context, req *in.SendFriendRequestRequest) (*out.SendFriendRequestResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	requestID := uuid.NewString()
	now := nowUTC()

	response := &out.SendFriendRequestResponse{}
	if err := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		pairAgg, err := txRepos.RelationshipPairAggregateRepository().LoadForUpdate(ctx, accountID, req.TargetUserID)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := pairAgg.SendFriendRequest(requestID, now); err != nil {
			return stackErr.Error(err)
		}
		friendRequest := pairAgg.FriendRequest()
		response.RequestID = friendRequest.AggregateID()
		response.RequesterID = friendRequest.RequesterID
		response.AddresseeID = friendRequest.AddresseeID
		response.Status = friendRequest.Status.String()

		if err := txRepos.RelationshipPairAggregateRepository().Save(ctx, pairAgg); err != nil {
			return stackErr.Error(err)
		}
		return nil
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return response, nil
}
