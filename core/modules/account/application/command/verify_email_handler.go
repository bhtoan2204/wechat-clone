package command

import (
	"context"
	"errors"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/support"
	repos "wechat-clone/core/modules/account/domain/repos"
	"wechat-clone/core/modules/account/domain/rules"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type verifyEmailHandler struct {
	baseRepo     repos.Repos
	verification emailVerificationDependencies
}

func NewVerifyEmailHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos) cqrs.Handler[*in.VerifyEmailRequest, *out.VerifyEmailResponse] {
	return &verifyEmailHandler{
		baseRepo:     baseRepo,
		verification: newEmailVerificationDependencies(appCtx),
	}
}

func (u *verifyEmailHandler) Handle(ctx context.Context, req *in.VerifyEmailRequest) (*out.VerifyEmailResponse, error) {
	_ = req
	log := logging.FromContext(ctx).Named("VerifyEmail")

	accountID, err := support.AccountIDFromCtx(ctx)
	if err != nil {
		log.Errorw("Failed to resolve account from context", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	accountAggregate, err := u.baseRepo.AccountAggregateRepository().Load(ctx, accountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = rules.ErrAccountNotFound
		}
		log.Errorw("Failed to load account aggregate", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	if err := accountAggregate.EnsureEmailVerificationAllowed(); err != nil {
		log.Errorw("Email verification is not allowed", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	accountEntity, err := accountAggregate.Snapshot()
	if err != nil {
		log.Errorw("Failed to build account projection", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	token, _, err := u.verification.SendVerificationEmail(ctx, accountEntity)
	if err != nil {
		log.Errorw("Failed to send verification email", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	if err := accountAggregate.RequestEmailVerification(token, time.Now().UTC()); err != nil {
		log.Errorw("Failed to record verification request", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		return txRepos.AccountAggregateRepository().Save(ctx, accountAggregate)
	}); txErr != nil {
		log.Errorw("Failed to publish verification requested event", zap.Error(txErr))
		return nil, stackErr.Error(txErr)
	}

	return &out.VerifyEmailResponse{
		Message: "Verification email queued",
	}, nil
}
