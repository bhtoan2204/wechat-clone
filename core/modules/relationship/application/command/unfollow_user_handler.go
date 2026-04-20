package command

import (
	"context"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/relationship/application/dto/in"
	"wechat-clone/core/modules/relationship/application/dto/out"
	repos "wechat-clone/core/modules/relationship/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type unfollowUserHandler struct {
	baseRepo repos.Repos
}

func NewUnfollowUser(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.UnfollowUserRequest, *out.UnfollowUserResponse] {
	return &unfollowUserHandler{
		baseRepo: baseRepo,
	}
}

func (u *unfollowUserHandler) Handle(ctx context.Context, req *in.UnfollowUserRequest) (*out.UnfollowUserResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		pairAgg, err := txRepos.RelationshipPairAggregateRepository().LoadForUpdate(ctx, accountID, req.TargetUserID)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := pairAgg.Unfollow(); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RelationshipPairAggregateRepository().Save(ctx, pairAgg); err != nil {
			return stackErr.Error(err)
		}
		return nil
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.UnfollowUserResponse{Success: true}, nil
}
