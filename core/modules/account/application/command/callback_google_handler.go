package command

import (
	"context"
	"errors"
	"fmt"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/application/dto/in"
	"wechat-clone/core/modules/account/application/dto/out"
	"wechat-clone/core/modules/account/application/provider"
	"wechat-clone/core/modules/account/domain/aggregate"
	"wechat-clone/core/modules/account/domain/entity"
	repos "wechat-clone/core/modules/account/domain/repos"
	valueobject "wechat-clone/core/modules/account/domain/value_object"
	"wechat-clone/core/shared/infra/xpaseto"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type callbackGoogleHandler struct {
	baseRepo             repos.Repos
	paseto               xpaseto.PasetoService
	authProviderRegistry *provider.AuthProviderRegistry
}

func NewCallbackGoogle(
	appCtx *appCtx.AppContext,
	baseRepo repos.Repos,
	authProviderRegistry *provider.AuthProviderRegistry,
) cqrs.Handler[*in.CallbackGoogleRequest, *out.CallbackGoogleResponse] {
	return &callbackGoogleHandler{
		baseRepo:             baseRepo,
		paseto:               appCtx.GetPaseto(),
		authProviderRegistry: authProviderRegistry,
	}
}

func (u *callbackGoogleHandler) Handle(ctx context.Context, req *in.CallbackGoogleRequest) (*out.CallbackGoogleResponse, error) {
	log := logging.FromContext(ctx)
	googleProvider, err := u.authProviderRegistry.Get("google")
	if err != nil {
		return nil, stackErr.Error(err)
	}
	callbackData, err := googleProvider.Callback(ctx, req.Code)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	userInfo, err := googleProvider.UserInfo(ctx, callbackData.AccessToken)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	email, err := valueobject.NewEmail(userInfo.Email)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	now := time.Now().UTC()
	var res out.CallbackGoogleResponse
	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		accountRepo := txRepos.AccountAggregateRepository()
		accountAgg, err := accountRepo.LoadByEmail(ctx, email.Value())
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return stackErr.Error(fmt.Errorf("load account aggregate by email: %w", err))
			}
			accountAgg, err = aggregate.NewAccountAggregate(uuid.NewString())
			if err != nil {
				return stackErr.Error(err)
			}
			displayName := userInfo.Name
			if displayName == "" {
				displayName = userInfo.Email
			}
			if err := accountAgg.OpenRegister(email.Value(), displayName, userInfo.Picture, now); err != nil {
				return stackErr.Error(err)
			}
			if err := accountRepo.Save(ctx, accountAgg); err != nil {
				return stackErr.Error(fmt.Errorf("save account: %w", err))
			}
		} else if !accountAgg.IsEmailVerified() {
			if err := accountAgg.ConfirmEmailVerified(email, now); err != nil {
				return stackErr.Error(err)
			}
			if err := accountRepo.Save(ctx, accountAgg); err != nil {
				return stackErr.Error(fmt.Errorf("save account: %w", err))
			}
		}

		snapshot, err := accountAgg.Snapshot()
		if err != nil {
			return stackErr.Error(err)
		}
		deviceReg := entity.DeviceRegistration{
			DeviceUID: req.DeviceUid, DeviceName: req.DeviceName, DeviceType: req.DeviceType,
			OSName: req.OsName, OSVersion: req.OsVersion, AppVersion: req.AppVersion,
			UserAgent: req.UserAgent, IPAddress: req.IpAddress,
		}
		deviceAgg, err := txRepos.DeviceAggregateRepository().FindByAccountAndUID(ctx, snapshot.ID, req.DeviceUid)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return stackErr.Error(fmt.Errorf("load device: %w", err))
			}
			deviceAgg, err = aggregate.NewDeviceAggregate(uuid.NewString())
			if err != nil {
				return stackErr.Error(err)
			}
			if err := deviceAgg.Register(snapshot.ID, deviceReg, now); err != nil {
				return stackErr.Error(err)
			}
		} else if err := deviceAgg.RefreshRegistration(deviceReg, now); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.DeviceAggregateRepository().Save(ctx, deviceAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save device: %w", err))
		}

		sessionID := uuid.NewString()
		accessToken, accessExp, refreshToken, refreshExp, err := u.issueTokenPair(ctx, u.paseto, *snapshot, xpaseto.RefreshTokenSubject{
			SessionID: sessionID,
			DeviceID:  deviceAgg.DeviceID(),
		})
		if err != nil {
			return stackErr.Error(err)
		}
		sessionAgg, err := aggregate.NewSessionAggregate(sessionID)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := sessionAgg.Create(snapshot.ID, deviceAgg.DeviceID(), refreshToken, refreshExp, now, req.IpAddress, req.UserAgent); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.SessionAggregateRepository().Save(ctx, sessionAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save session: %w", err))
		}

		res = out.CallbackGoogleResponse{
			AccessToken:      accessToken,
			AccessExpiresAt:  accessExp.UnixMilli(),
			RefreshToken:     refreshToken,
			RefreshExpiresAt: refreshExp.UnixMilli(),
		}
		return nil
	}); txErr != nil {
		log.Errorw("Google login failed", zap.Error(txErr), zap.Any("userInfo", userInfo))
		return nil, stackErr.Error(txErr)
	}

	return &res, nil
}

func (u *callbackGoogleHandler) issueTokenPair(
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
