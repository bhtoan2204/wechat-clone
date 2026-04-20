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

type getRelationshipSummaryHandler struct {
}

func NewGetRelationshipSummary(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
) cqrs.Handler[*in.GetRelationshipSummaryRequest, *out.GetRelationshipSummaryResponse] {
	return &getRelationshipSummaryHandler{}
}

func (u *getRelationshipSummaryHandler) Handle(ctx context.Context, req *in.GetRelationshipSummaryRequest) (*out.GetRelationshipSummaryResponse, error) {
	return nil, fmt.Errorf("not implemented yet")
}
