// CODE_GENERATOR: module-repository
package repository

import (
	"context"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/shared/pkg/stackErr"

	"wechat-clone/core/modules/relationship/domain/repos"
)

type repoImpl struct {
	appCtx *appCtx.AppContext
}

func NewRepoImpl(appCtx *appCtx.AppContext) repos.Repos {
	return &repoImpl{appCtx: appCtx}
}

func (r *repoImpl) WithTransaction(_ context.Context, fn func(repos.Repos) error) error {
	if fn == nil {
		return nil
	}
	return stackErr.Error(fn(r))
}
