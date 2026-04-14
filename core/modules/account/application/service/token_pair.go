package service

import (
	"context"
	"fmt"

	"go-socket/core/modules/account/domain/entity"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/stackErr"
)

func issueTokenPair(
	ctx context.Context,
	pasetoSvc xpaseto.PasetoService,
	account *entity.Account,
	subject xpaseto.RefreshTokenSubject,
) (*TokenPairResult, error) {
	if pasetoSvc == nil {
		return nil, stackErr.Error(fmt.Errorf("paseto service is required"))
	}
	if account == nil {
		return nil, stackErr.Error(fmt.Errorf("account snapshot is required"))
	}

	accessToken, accessExpiresAt, err := pasetoSvc.GenerateAccessToken(ctx, account)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("generate access token failed: %v", err))
	}

	refreshToken, refreshExpiresAt, err := pasetoSvc.GenerateRefreshToken(ctx, account, subject)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("generate refresh token failed: %v", err))
	}

	return &TokenPairResult{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExpiresAt,
	}, nil
}
