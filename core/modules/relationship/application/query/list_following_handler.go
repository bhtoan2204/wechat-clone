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

type listFollowingHandler struct {
	projRepo relationshipprojection.ReadRepository
}

func NewListFollowing(
	appCtx *appCtx.AppContext,
	projRepo relationshipprojection.ReadRepository,
) cqrs.Handler[*in.ListFollowingRequest, *out.ListFollowingResponse] {
	return &listFollowingHandler{projRepo: projRepo}
}

func (u *listFollowingHandler) Handle(ctx context.Context, req *in.ListFollowingRequest) (*out.ListFollowingResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	result, err := u.projRepo.ListFollowing(ctx, normalizeListTarget(accountID, req.UserID), req.Cursor, normalizeLimit(req.Limit))
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if result == nil {
		result = emptyListResult()
	}
	return &out.ListFollowingResponse{Items: result.Items, NextCursor: result.NextCursor, Total: result.Total}, nil
}
