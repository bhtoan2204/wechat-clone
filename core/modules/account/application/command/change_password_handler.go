package command

import (
	"context"

	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/application/service"
	"go-socket/core/modules/account/application/support"
	repos "go-socket/core/modules/account/domain/repos"
	domainservice "go-socket/core/modules/account/domain/service"
	valueobject "go-socket/core/modules/account/domain/value_object"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/hasher"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"go-socket/core/shared/utils"

	"go.uber.org/zap"
)

type changePasswordHandler struct {
	baseRepo repos.Repos
	hasher   hasher.Hasher
}

func NewChangePasswordHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos, services service.Services) cqrs.Handler[*in.ChangePasswordRequest, *out.ChangePasswordResponse] {
	return &changePasswordHandler{
		baseRepo: baseRepo,
		hasher:   appCtx.GetHasher(),
	}
}

func (u *changePasswordHandler) Handle(ctx context.Context, req *in.ChangePasswordRequest) (*out.ChangePasswordResponse, error) {
	log := logging.FromContext(ctx).Named("ChangePassword")

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

	currentHash, err := accountAggregate.CurrentPasswordHash()
	if err != nil {
		log.Errorw("Failed to resolve current password hash", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	valid, err := u.hasher.Verify(ctx, req.CurrentPassword, currentHash.Value())
	if err != nil {
		log.Errorw("Failed to verify current password", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	if !valid {
		return nil, stackErr.Error(ErrInvalidCurrentPassword)
	}

	newPassword, err := valueobject.NewPlainPassword(req.NewPassword)
	if err != nil {
		log.Errorw("Failed to validate new password", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	if err := domainservice.EnsurePasswordIsNew(ctx, u.hasher, newPassword, currentHash); err != nil {
		log.Errorw("Failed password reuse policy", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	hashedPassword, err := u.hasher.Hash(ctx, newPassword.Value())
	if err != nil {
		log.Errorw("Failed to hash new password", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	hashedPasswordVO, err := valueobject.NewHashedPassword(hashedPassword)
	if err != nil {
		log.Errorw("Failed to create password value object", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	now := utils.NowUTC()
	changed, err := accountAggregate.ChangePassword(hashedPasswordVO, now)
	if err != nil {
		log.Errorw("Failed to change password on aggregate", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	if !changed {
		return &out.ChangePasswordResponse{Message: "Password is unchanged"}, nil
	}

	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.AccountAggregateRepository().Save(ctx, accountAggregate); err != nil {
			return stackErr.Error(err)
		}
		sessionAggs, err := txRepos.SessionRepository().ListByAccountID(ctx, accountID)
		if err != nil {
			return stackErr.Error(err)
		}
		for _, sessionAgg := range sessionAggs {
			changed, err := sessionAgg.Revoke("password_changed", now)
			if err != nil {
				return stackErr.Error(err)
			}
			if !changed {
				continue
			}
			if err := txRepos.SessionRepository().Save(ctx, sessionAgg); err != nil {
				return stackErr.Error(err)
			}
		}
		return nil
	}); txErr != nil {
		log.Errorw("Failed to persist changed password", zap.Error(txErr))
		return nil, stackErr.Error(txErr)
	}

	return &out.ChangePasswordResponse{
		Message: "Password changed successfully",
	}, nil
}
