package command

import (
	"context"
	"errors"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/service"
	"wechat-clone/core/modules/account/application/support"
	repos "wechat-clone/core/modules/account/domain/repos"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type updateProfileHandler struct {
	baseRepo repos.Repos
}

func NewUpdateProfileHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos, services service.Services) cqrs.Handler[*in.UpdateProfileRequest, *out.UpdateProfileResponse] {
	return &updateProfileHandler{
		baseRepo: baseRepo,
	}
}

func (u *updateProfileHandler) Handle(ctx context.Context, req *in.UpdateProfileRequest) (*out.UpdateProfileResponse, error) {
	log := logging.FromContext(ctx).Named("UpdateProfile")

	accountID, err := support.AccountIDFromCtx(ctx)
	if err != nil {
		log.Errorw("Failed to resolve account from context", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	accountAggregate, err := u.baseRepo.AccountAggregateRepository().Load(ctx, accountID)
	if err != nil {
		log.Errorw("Failed to load account aggregate", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	updated, err := accountAggregate.UpdateProfile(req.DisplayName, req.Username, req.AvatarObjectKey, time.Now().UTC())
	if err != nil {
		log.Errorw("Failed to update account profile aggregate", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	if !updated {
		accountEntity, err := accountAggregate.Snapshot()
		if err != nil {
			return nil, stackErr.Error(err)
		}
		return support.ToUpdateProfileResponse(accountEntity), nil
	}

	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		return txRepos.AccountAggregateRepository().Save(ctx, accountAggregate)
	}); txErr != nil {
		if errors.Is(txErr, repos.ErrAccountUsernameAlreadyExists) {
			return nil, stackErr.Error(ErrUsernameExists)
		}
		log.Errorw("Failed to persist updated profile", zap.Error(txErr))
		return nil, stackErr.Error(txErr)
	}

	accountEntity, err := accountAggregate.Snapshot()
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return support.ToUpdateProfileResponse(accountEntity), nil
}
