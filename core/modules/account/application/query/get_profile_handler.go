package query

import (
	"context"
	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/projection"
	"wechat-clone/core/modules/account/application/support"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type getProfileHandler struct {
	accountReadRepo projection.AccountReadRepository
}

func NewGetProfileHandler(appCtx *appCtx.AppContext, accountReadRepo projection.AccountReadRepository) cqrs.Handler[*in.GetProfileRequest, *out.GetProfileResponse] {
	return &getProfileHandler{
		accountReadRepo: accountReadRepo,
	}
}

func (u *getProfileHandler) Handle(ctx context.Context, req *in.GetProfileRequest) (*out.GetProfileResponse, error) {
	_ = req
	log := logging.FromContext(ctx).Named("GetProfile")
	accountID, err := support.AccountIDFromCtx(ctx)
	if err != nil {
		log.Errorw("Account not found in context", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	accountEntity, err := u.accountReadRepo.GetAccountByID(ctx, accountID)
	if err != nil {
		log.Errorw("Failed to get account by ID", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	return support.ToGetProfileResponse(accountEntity), nil
}
