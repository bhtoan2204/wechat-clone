package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/domain/aggregate"
	"wechat-clone/core/modules/account/domain/entity"
	repos "wechat-clone/core/modules/account/domain/repos"
	valueobject "wechat-clone/core/modules/account/domain/value_object"
	"wechat-clone/core/shared/infra/xpaseto"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/hasher"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

type registerHandler struct {
	baseRepo repos.Repos
	hasher   hasher.Hasher
	paseto   xpaseto.PasetoService
}

func NewRegisterHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos) cqrs.Handler[*in.RegisterRequest, *out.RegisterResponse] {
	return &registerHandler{
		baseRepo: baseRepo,
		hasher:   appCtx.GetHasher(),
		paseto:   appCtx.GetPaseto(),
	}
}

func (u *registerHandler) Handle(ctx context.Context, req *in.RegisterRequest) (*out.RegisterResponse, error) {
	now := time.Now().UTC()

	email, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	password, err := valueobject.NewPlainPassword(req.Password)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	hashedPassword, err := u.hasher.Hash(ctx, password.Value())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	hashedPasswordVO, err := valueobject.NewHashedPassword(hashedPassword)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountAgg, err := aggregate.NewAccountAggregate(uuid.NewString())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	var res out.RegisterResponse
	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := accountAgg.Register(email, hashedPasswordVO, req.DisplayName, now); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.AccountAggregateRepository().Save(ctx, accountAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save account: %w", err))
		}
		deviceAgg, err := aggregate.NewDeviceAggregate(uuid.NewString())
		if err != nil {
			return stackErr.Error(err)
		}
		deviceReg := entity.DeviceRegistration{
			DeviceUID: req.DeviceUid, DeviceName: req.DeviceName, DeviceType: req.DeviceType,
			OSName: req.OsName, OSVersion: req.OsVersion, AppVersion: req.AppVersion,
			UserAgent: req.UserAgent, IPAddress: req.IpAddress,
		}
		if err := deviceAgg.Register(accountAgg.AggregateID(), deviceReg, now); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.DeviceAggregateRepository().Save(ctx, deviceAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save device: %w", err))
		}
		sessionID := uuid.NewString()
		subject := xpaseto.RefreshTokenSubject{
			SessionID: sessionID,
			DeviceID:  deviceAgg.DeviceID(),
		}

		snapshot, _ := accountAgg.Snapshot()
		accessToken, accessExp, refreshToken, refreshExp, err := u.issueTokenPair(ctx, u.paseto, *snapshot, subject)
		if err != nil {
			return stackErr.Error(err)
		}

		refreshTokenHash, err := u.hasher.Hash(ctx, refreshToken)
		if err != nil {
			return stackErr.Error(err)
		}

		sessionAgg, err := aggregate.NewSessionAggregate(sessionID)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := sessionAgg.Create(
			accountAgg.AggregateID(), deviceAgg.DeviceID(),
			refreshTokenHash, refreshExp, now, req.IpAddress, req.UserAgent,
		); err != nil {
			return stackErr.Error(err)
		}

		if err := txRepos.SessionAggregateRepository().Save(ctx, sessionAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save session: %w", err))
		}

		res = out.RegisterResponse{
			AccessToken:      accessToken,
			AccessExpiresAt:  accessExp.UnixMilli(),
			RefreshToken:     refreshToken,
			RefreshExpiresAt: refreshExp.UnixMilli(),
		}
		return nil
	}); txErr != nil {
		if errors.Is(txErr, repos.ErrAccountEmailAlreadyExists) {
			return nil, stackErr.Error(ErrRegistrationAccountExists)
		}
		return nil, stackErr.Error(txErr)
	}

	return &res, nil
}

func (u *registerHandler) issueTokenPair(
	ctx context.Context,
	pasetoSvc xpaseto.PasetoService,
	account entity.Account,
	subject xpaseto.RefreshTokenSubject,
) (string, time.Time, string, time.Time, error) {
	if account.ID == "" {
		return "", time.Time{}, "", time.Time{}, stackErr.Error(fmt.Errorf("account snapshot is required"))
	}

	accessToken, accessExpiresAt, err := pasetoSvc.GenerateAccessToken(ctx, &account)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, stackErr.Error(fmt.Errorf("generate access token failed: %w", err))
	}

	refreshToken, refreshExpiresAt, err := pasetoSvc.GenerateRefreshToken(ctx, &account, subject)
	if err != nil {
		return "", time.Time{}, "", time.Time{}, stackErr.Error(fmt.Errorf("generate refresh token failed: %w", err))
	}

	return accessToken, accessExpiresAt, refreshToken, refreshExpiresAt, nil
}
