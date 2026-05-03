package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/domain/entity"
	repos "wechat-clone/core/modules/account/domain/repos"
	"wechat-clone/core/shared/infra/xpaseto"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/hasher"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

func mapRefreshSessionErr(err error) error {
	switch {
	case errors.Is(err, entity.ErrSessionExpired):
		return ErrRefreshSessionExpired
	case errors.Is(err, entity.ErrSessionRevoked):
		return ErrRefreshSessionRevoked
	default:
		return ErrRefreshTokenInvalid
	}
}

type refreshHandler struct {
	baseRepo repos.Repos
	hasher   hasher.Hasher
	paseto   xpaseto.PasetoService
}

func NewRefresh(appCtx *appCtx.AppContext, baseRepo repos.Repos) cqrs.Handler[*in.RefreshRequest, *out.RefreshResponse] {
	return &refreshHandler{
		baseRepo: baseRepo,
		hasher:   appCtx.GetHasher(),
		paseto:   appCtx.GetPaseto(),
	}
}

func (u *refreshHandler) Handle(ctx context.Context, req *in.RefreshRequest) (*out.RefreshResponse, error) {
	claims, err := u.paseto.ParseRefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%v: %w", ErrRefreshTokenInvalid, err))
	}

	now := time.Now().UTC()
	var res out.RefreshResponse
	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		sessionAgg, err := txRepos.SessionAggregateRepository().Load(ctx, claims.SessionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return stackErr.Error(ErrRefreshTokenInvalid)
			}
			return stackErr.Error(fmt.Errorf("load session: %w", err))
		}
		session, err := sessionAgg.Snapshot()
		if err != nil {
			return stackErr.Error(err)
		}
		if session.AccountID != claims.AccountID || session.DeviceID != claims.DeviceID {
			return stackErr.Error(ErrRefreshTokenInvalid)
		}

		valid, err := u.hasher.Verify(ctx, req.RefreshToken, session.RefreshTokenHash)
		if err != nil {
			return stackErr.Error(fmt.Errorf("verify refresh token: %w", err))
		}
		if !valid {
			return stackErr.Error(ErrRefreshTokenInvalid)
		}

		if err := sessionAgg.EnsureRefreshAllowed(now); err != nil {
			if errors.Is(err, entity.ErrSessionExpired) && sessionAgg.MarkExpired(now) {
				if saveErr := txRepos.SessionAggregateRepository().Save(ctx, sessionAgg); saveErr != nil {
					return stackErr.Error(fmt.Errorf("mark session expired: %w", saveErr))
				}
			}
			return stackErr.Error(mapRefreshSessionErr(err))
		}

		accountAgg, err := txRepos.AccountAggregateRepository().Load(ctx, claims.AccountID)
		if err != nil {
			return stackErr.Error(fmt.Errorf("load account aggregate: %w", err))
		}
		accountSnapshot, err := accountAgg.Snapshot()
		if err != nil {
			return stackErr.Error(err)
		}

		deviceAgg, err := txRepos.DeviceAggregateRepository().GetByAccountAndID(ctx, claims.AccountID, session.DeviceID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return stackErr.Error(ErrRefreshTokenInvalid)
			}
			return stackErr.Error(fmt.Errorf("load device: %w", err))
		}
		if err := deviceAgg.Touch(req.UserAgent, req.IpAddress, now); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.DeviceAggregateRepository().Save(ctx, deviceAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save device: %w", err))
		}

		tokenPair, err := issueAccountTokenPair(ctx, u.paseto, *accountSnapshot, xpaseto.RefreshTokenSubject{
			SessionID: sessionAgg.SessionID(),
			DeviceID:  sessionAgg.DeviceID(),
		})
		if err != nil {
			return stackErr.Error(err)
		}
		refreshTokenHash, err := u.hasher.Hash(ctx, tokenPair.refreshToken)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := sessionAgg.Rotate(refreshTokenHash, tokenPair.refreshExpiresAt, now, req.IpAddress, req.UserAgent); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.SessionAggregateRepository().Save(ctx, sessionAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save session: %w", err))
		}

		res = out.RefreshResponse{
			AccessToken:      tokenPair.accessToken,
			AccessExpiresAt:  tokenPair.accessExpiresAt.UnixMilli(),
			RefreshToken:     tokenPair.refreshToken,
			RefreshExpiresAt: tokenPair.refreshExpiresAt.UnixMilli(),
		}
		return nil
	}); txErr != nil {
		return nil, stackErr.Error(txErr)
	}

	return &res, nil
}
