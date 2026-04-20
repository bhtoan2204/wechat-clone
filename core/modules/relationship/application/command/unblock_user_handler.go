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

type unblockUserHandler struct {
	baseRepo repos.Repos
}

func NewUnblockUser(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.UnblockUserRequest, *out.UnblockUserResponse] {
	return &unblockUserHandler{
		baseRepo: baseRepo,
	}
}

func (u *unblockUserHandler) Handle(ctx context.Context, req *in.UnblockUserRequest) (*out.UnblockUserResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		pairAgg, err := txRepos.RelationshipPairAggregateRepository().LoadForUpdate(ctx, accountID, req.TargetUserID)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := pairAgg.Unblock(); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.RelationshipPairAggregateRepository().Save(ctx, pairAgg); err != nil {
			return stackErr.Error(err)
		}
		return nil
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.UnblockUserResponse{Success: true}, nil
}
