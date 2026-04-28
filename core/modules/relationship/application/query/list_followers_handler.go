package query

import (
	"context"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/relationship/application/dto/in"
	"wechat-clone/core/modules/relationship/application/dto/out"
	relationshipprojection "wechat-clone/core/modules/relationship/application/projection"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type listFollowersHandler struct {
	projRepo    relationshipprojection.ReadRepository
	accountRepo AccountReadRepository
}

func NewListFollowers(
	appCtx *appCtx.AppContext,
	projRepo relationshipprojection.ReadRepository,
	accountRepo AccountReadRepository,
) cqrs.Handler[*in.ListFollowersRequest, *out.ListFollowersResponse] {
	return &listFollowersHandler{projRepo: projRepo, accountRepo: accountRepo}
}

func (u *listFollowersHandler) Handle(ctx context.Context, req *in.ListFollowersRequest) (*out.ListFollowersResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	result, err := u.projRepo.ListFollowers(ctx, normalizeListTarget(accountID, req.UserID), req.Cursor, normalizeLimit(req.Limit))
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if result == nil {
		result = emptyListResult()
	}
	items, err := mapRelationshipAccountSummaries(ctx, u.accountRepo, result.Items)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return &out.ListFollowersResponse{Items: items, NextCursor: result.NextCursor, Total: result.Total}, nil
}
