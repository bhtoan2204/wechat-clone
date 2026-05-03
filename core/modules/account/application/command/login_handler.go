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
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type loginHandler struct {
	baseRepo repos.Repos
	hasher   hasher.Hasher
	paseto   xpaseto.PasetoService
}

func NewLoginHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos) cqrs.Handler[*in.LoginRequest, *out.LoginResponse] {
	return &loginHandler{
		baseRepo: baseRepo,
		hasher:   appCtx.GetHasher(),
		paseto:   appCtx.GetPaseto(),
	}
}

func (u *loginHandler) Handle(ctx context.Context, req *in.LoginRequest) (*out.LoginResponse, error) {
	log := logging.FromContext(ctx).Named("Login")
	now := time.Now().UTC()

	email, err := valueobject.NewEmail(req.Email)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	password, err := valueobject.NewPlainPassword(req.Password)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountAgg, err := u.baseRepo.AccountAggregateRepository().LoadByEmail(ctx, email.Value())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorw("Account not found", zap.String("email", req.Email))
			return nil, stackErr.Error(ErrAccountNotFound)
		}
		log.Errorw("Login failed", zap.Error(err), zap.String("email", req.Email))
		return nil, stackErr.Error(fmt.Errorf("load account aggregate by email failed: %w", err))
	}

	currentHash, err := accountAgg.CurrentPasswordHash()
	if err != nil {
		return nil, stackErr.Error(err)
	}
	valid, err := u.hasher.Verify(ctx, password.Value(), currentHash.Value())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if !valid {
		log.Errorw("Invalid credentials", zap.String("email", req.Email))
		return nil, stackErr.Error(ErrInvalidCredentials)
	}

	snapshot, err := accountAgg.Snapshot()
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var res out.LoginResponse
	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		deviceAgg, err := txRepos.DeviceAggregateRepository().GetByAccountAndID(ctx, snapshot.ID, req.DeviceUid)
		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return stackErr.Error(fmt.Errorf("load device: %w", err))
			}
			deviceAgg, err = aggregate.NewDeviceAggregate(uuid.NewString())
			if err != nil {
				return stackErr.Error(err)
			}
			deviceReg := entity.DeviceRegistration{
				DeviceUID: req.DeviceUid, DeviceName: req.DeviceName, DeviceType: req.DeviceType,
				OSName: req.OsName, OSVersion: req.OsVersion, AppVersion: req.AppVersion,
				UserAgent: req.UserAgent, IPAddress: req.IpAddress,
			}
			if err := deviceAgg.Register(snapshot.ID, deviceReg, now); err != nil {
				return stackErr.Error(err)
			}
		} else {
			deviceReg := entity.DeviceRegistration{
				DeviceUID: req.DeviceUid, DeviceName: req.DeviceName, DeviceType: req.DeviceType,
				OSName: req.OsName, OSVersion: req.OsVersion, AppVersion: req.AppVersion,
				UserAgent: req.UserAgent, IPAddress: req.IpAddress,
			}
			if err := deviceAgg.RefreshRegistration(deviceReg, now); err != nil {
				return stackErr.Error(err)
			}
		}
		if err := txRepos.DeviceAggregateRepository().Save(ctx, deviceAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save device: %w", err))
		}

		sessionID := uuid.NewString()
		tokenPair, err := issueAccountTokenPair(ctx, u.paseto, *snapshot, xpaseto.RefreshTokenSubject{
			SessionID: sessionID,
			DeviceID:  deviceAgg.DeviceID(),
		})
		if err != nil {
			return stackErr.Error(err)
		}
		refreshTokenHash, err := u.hasher.Hash(ctx, tokenPair.refreshToken)
		if err != nil {
			return stackErr.Error(err)
		}

		sessionAgg, err := aggregate.NewSessionAggregate(sessionID)
		if err != nil {
			return stackErr.Error(err)
		}
		if err := sessionAgg.Create(snapshot.ID, deviceAgg.DeviceID(), refreshTokenHash, tokenPair.refreshExpiresAt, now, req.IpAddress, req.UserAgent); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.SessionAggregateRepository().Save(ctx, sessionAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save session: %w", err))
		}

		res = out.LoginResponse{
			AccessToken:      tokenPair.accessToken,
			AccessExpiresAt:  tokenPair.accessExpiresAt.UnixMilli(),
			RefreshToken:     tokenPair.refreshToken,
			RefreshExpiresAt: tokenPair.refreshExpiresAt.UnixMilli(),
		}
		return nil
	}); txErr != nil {
		log.Errorw("Login failed", zap.Error(txErr), zap.String("email", req.Email))
		return nil, stackErr.Error(txErr)
	}

	return &res, nil
}
