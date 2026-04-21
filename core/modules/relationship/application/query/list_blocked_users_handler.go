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

type listBlockedUsersHandler struct {
	projRepo relationshipprojection.ReadRepository
}

func NewListBlockedUsers(
	appCtx *appCtx.AppContext,
	projRepo relationshipprojection.ReadRepository,
) cqrs.Handler[*in.ListBlockedUsersRequest, *out.ListBlockedUsersResponse] {
	return &listBlockedUsersHandler{projRepo: projRepo}
}

func (u *listBlockedUsersHandler) Handle(ctx context.Context, req *in.ListBlockedUsersRequest) (*out.ListBlockedUsersResponse, error) {
	accountID, err := currentAccountID(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	result, err := u.projRepo.ListBlockedUsers(ctx, accountID, req.Cursor, normalizeLimit(req.Limit))
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if result == nil {
		result = emptyListResult()
	}
	return &out.ListBlockedUsersResponse{Items: result.Items, NextCursor: result.NextCursor, Total: result.Total}, nil
}
