package query

import (
	"context"
	"fmt"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/relationship/application/dto/in"
	"wechat-clone/core/modules/relationship/application/dto/out"
	repos "wechat-clone/core/modules/relationship/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
)

type listBlockedUsersHandler struct {
}

func NewListBlockedUsers(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.ListBlockedUsersRequest, *out.ListBlockedUsersResponse] {
	return &listBlockedUsersHandler{}
}

func (u *listBlockedUsersHandler) Handle(ctx context.Context, req *in.ListBlockedUsersRequest) (*out.ListBlockedUsersResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
