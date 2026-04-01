package query

import (
	"context"
	"errors"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	repos "go-socket/core/modules/account/domain/repos"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"time"

	"go.uber.org/zap"
)

type getProfileHandler struct {
	accountRepo repos.AccountRepository
}

func NewGetProfileHandler(baseRepo repos.Repos) cqrs.Handler[*in.GetProfileRequest, *out.GetProfileResponse] {
	return &getProfileHandler{
		accountRepo: baseRepo.AccountRepository(),
	}
}

func (u *getProfileHandler) Handle(ctx context.Context, req *in.GetProfileRequest) (*out.GetProfileResponse, error) {
	_ = req
	log := logging.FromContext(ctx).Named("GetProfile")
	account := ctx.Value("account")
	if account == nil {
		log.Errorw("Account not found", zap.Error(errors.New("account not found")))
		return nil, stackerr.Error(errors.New("account not found"))
	}

	payload, ok := account.(*xpaseto.PasetoPayload)
	if !ok {
		return nil, stackerr.Error(errors.New("invalid account payload"))
	}

	accountEntity, err := u.accountRepo.GetAccountByID(ctx, payload.AccountID)
	if err != nil {
		log.Errorw("Failed to get account by ID", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	return &out.GetProfileResponse{
		Email:     accountEntity.Email.Value(),
		CreatedAt: accountEntity.CreatedAt.Format(time.RFC3339),
		UpdatedAt: accountEntity.UpdatedAt.Format(time.RFC3339),
	}, nil
}
