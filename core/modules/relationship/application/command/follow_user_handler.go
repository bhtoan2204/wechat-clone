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

type followUserHandler struct {
	baseRepo repos.Repos
}

func NewFollowUser(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.FollowUserRequest, *out.FollowUserResponse] {
	return &followUserHandler{
		baseRepo: baseRepo,
	}
}

func (u *followUserHandler) Handle(ctx context.Context, req *in.FollowUserRequest) (*out.FollowUserResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	now := nowUTC()
	response := &out.FollowUserResponse{}
	if err := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		pairAgg, err := txRepos.RelationshipPairAggregateRepository().LoadForUpdate(ctx, accountID, req.TargetUserID)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := pairAgg.Follow(uuid.NewString(), now); err != nil {
			return stackErr.Error(err)
		}

		relation := pairAgg.FollowCreated()
		response.FollowerID = relation.FollowerID
		response.FolloweeID = relation.FolloweeID
		response.CreatedAt = relation.CreatedAt.Unix()

		if err := txRepos.RelationshipPairAggregateRepository().Save(ctx, pairAgg); err != nil {
			return stackErr.Error(err)
		}
		return nil
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return response, nil
}
