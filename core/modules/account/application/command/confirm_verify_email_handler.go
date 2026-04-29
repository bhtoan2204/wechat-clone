package command

import (
	"context"
	"errors"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	repos "wechat-clone/core/modules/account/domain/repos"
	"wechat-clone/core/modules/account/domain/rules"
	valueobject "wechat-clone/core/modules/account/domain/value_object"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type confirmVerifyEmailHandler struct {
	baseRepo     repos.Repos
	verification emailVerificationDependencies
}

func NewConfirmVerifyEmailHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos) cqrs.Handler[*in.ConfirmVerifyEmailRequest, *out.ConfirmVerifyEmailResponse] {
	return &confirmVerifyEmailHandler{
		baseRepo:     baseRepo,
		verification: newEmailVerificationDependencies(appCtx),
	}
}

func (u *confirmVerifyEmailHandler) Handle(ctx context.Context, req *in.ConfirmVerifyEmailRequest) (*out.ConfirmVerifyEmailResponse, error) {
	log := logging.FromContext(ctx).Named("ConfirmVerifyEmail")

	tokenPayload, err := u.verification.ConsumeVerificationToken(ctx, req.Token)
	if err != nil {
		log.Errorw("Failed to consume verification token", zap.Error(err))
		return nil, stackErr.Error(ErrInvalidVerificationToken)
	}

	accountAggregate, err := u.baseRepo.AccountAggregateRepository().Load(ctx, tokenPayload.AccountID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			err = rules.ErrAccountNotFound
		}
		log.Errorw("Failed to load account aggregate", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	tokenEmail, err := valueobject.NewEmail(tokenPayload.Email)
	if err != nil {
		return nil, stackErr.Error(ErrInvalidVerificationToken)
	}

	err = accountAggregate.ConfirmEmailVerified(tokenEmail, utils.NowUTC())
	if err != nil {
		if errors.Is(err, rules.ErrAccountEmailMismatch) {
			return nil, stackErr.Error(ErrInvalidVerificationToken)
		}
		log.Errorw("Failed to confirm email verification", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		return txRepos.AccountAggregateRepository().Save(ctx, accountAggregate)
	}); txErr != nil {
		log.Errorw("Failed to persist verified email", zap.Error(txErr))
		return nil, stackErr.Error(txErr)
	}

	accountEntity, err := accountAggregate.Snapshot()
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return &out.ConfirmVerifyEmailResponse{
		Message:    "Email verified successfully",
		VerifiedAt: accountEntity.EmailVerifiedAt.UTC().Format(time.RFC3339),
	}, nil
}
