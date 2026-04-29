package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/support"
	repos "wechat-clone/core/modules/account/domain/repos"
	"wechat-clone/core/shared/infra/xpaseto"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type logoutHandler struct {
	baseRepo repos.Repos
	paseto   xpaseto.PasetoService
}

func NewLogoutHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos) cqrs.Handler[*in.LogoutRequest, *out.LogoutResponse] {
	return &logoutHandler{
		baseRepo: baseRepo,
		paseto:   appCtx.GetPaseto(),
	}
}

func (u *logoutHandler) Handle(ctx context.Context, req *in.LogoutRequest) (*out.LogoutResponse, error) {
	accountID, err := support.AccountIDFromCtx(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if accountID == "" {
		return nil, stackErr.Error(ErrRefreshTokenInvalid)
	}

	now := time.Now().UTC()
	if req.Token == "" {
		if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
			sessionAggs, err := txRepos.SessionAggregateRepository().ListByAccountID(ctx, accountID)
			if err != nil {
				return stackErr.Error(err)
			}
			for _, sessionAgg := range sessionAggs {
				changed, err := sessionAgg.Revoke("logout_all", now)
				if err != nil {
					return stackErr.Error(err)
				}
				if !changed {
					continue
				}
				if err := txRepos.SessionAggregateRepository().Save(ctx, sessionAgg); err != nil {
					return stackErr.Error(err)
				}
			}
			return nil
		}); txErr != nil {
			return nil, stackErr.Error(txErr)
		}
		return &out.LogoutResponse{Message: "Logout successful"}, nil
	}

	claims, err := u.paseto.ParseRefreshToken(ctx, req.Token)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if claims.AccountID != accountID {
		return nil, stackErr.Error(ErrRefreshTokenInvalid)
	}

	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		sessionAgg, err := txRepos.SessionAggregateRepository().Load(ctx, claims.SessionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return stackErr.Error(fmt.Errorf("load session: %w", err))
		}
		session, err := sessionAgg.Snapshot()
		if err != nil {
			return stackErr.Error(err)
		}
		if session.AccountID != accountID || session.DeviceID != claims.DeviceID {
			return stackErr.Error(ErrRefreshTokenInvalid)
		}
		changed, err := sessionAgg.Revoke("logout", now)
		if err != nil {
			return stackErr.Error(err)
		}
		if !changed {
			return nil
		}
		if err := txRepos.SessionAggregateRepository().Save(ctx, sessionAgg); err != nil {
			return stackErr.Error(err)
		}
		return nil
	}); txErr != nil {
		return nil, stackErr.Error(txErr)
	}

	return &out.LogoutResponse{Message: "Logout successful"}, nil
}
