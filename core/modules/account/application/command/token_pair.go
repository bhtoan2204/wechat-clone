package command

import (
	"context"
	"fmt"
	"time"

	"wechat-clone/core/modules/account/domain/entity"
	"wechat-clone/core/shared/infra/xpaseto"
	"wechat-clone/core/shared/pkg/stackErr"

	"golang.org/x/sync/errgroup"
)

type issuedTokenPair struct {
	accessToken      string
	accessExpiresAt  time.Time
	refreshToken     string
	refreshExpiresAt time.Time
}

func issueAccountTokenPair(
	ctx context.Context,
	pasetoSvc xpaseto.PasetoService,
	account entity.Account,
	subject xpaseto.RefreshTokenSubject,
) (*issuedTokenPair, error) {
	if account.ID == "" {
		return nil, stackErr.Error(fmt.Errorf("account snapshot is required"))
	}

	var pair issuedTokenPair
	eg, egCtx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		accessToken, accessExpiresAt, err := pasetoSvc.GenerateAccessToken(egCtx, &account)
		if err != nil {
			return stackErr.Error(fmt.Errorf("generate access token failed: %w", err))
		}
		pair.accessToken = accessToken
		pair.accessExpiresAt = accessExpiresAt
		return nil
	})
	eg.Go(func() error {
		refreshToken, refreshExpiresAt, err := pasetoSvc.GenerateRefreshToken(egCtx, &account, subject)
		if err != nil {
			return stackErr.Error(fmt.Errorf("generate refresh token failed: %w", err))
		}
		pair.refreshToken = refreshToken
		pair.refreshExpiresAt = refreshExpiresAt
		return nil
	})
	if err := eg.Wait(); err != nil {
		return nil, stackErr.Error(err)
	}

	return &pair, nil
}
