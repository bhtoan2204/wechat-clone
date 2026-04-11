package command

import (
	"context"
	"errors"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/application/service"
	"go-socket/core/modules/account/domain/aggregate"
	repos "go-socket/core/modules/account/domain/repos"
	domainservice "go-socket/core/modules/account/domain/service"
	valueobject "go-socket/core/modules/account/domain/value_object"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/hasher"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type registerHandler struct {
	baseRepo repos.Repos
	hasher   hasher.Hasher
	paseto   xpaseto.PasetoService
}

func NewRegisterHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos, services service.Services) cqrs.Handler[*in.RegisterRequest, *out.RegisterResponse] {
	return &registerHandler{
		baseRepo: baseRepo,
		hasher:   appCtx.GetHasher(),
		paseto:   appCtx.GetPaseto(),
	}
}

func (u *registerHandler) Handle(ctx context.Context, req *in.RegisterRequest) (*out.RegisterResponse, error) {
	log := logging.FromContext(ctx).Named("Register")
	accountRepo := u.baseRepo.AccountRepository()

	email, err := valueobject.NewEmail(req.Email)
	if err != nil {
		log.Errorw("Failed to create email", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	if err := domainservice.EnsureEmailAvailable(ctx, accountRepo, email); err != nil {
		if errors.Is(err, domainservice.ErrAccountEmailAlreadyExists) {
			log.Errorw("Account already exists", zap.String("email", email.Value()))
			return nil, stackErr.Error(ErrAccountExists)
		}
		log.Errorw("Failed to check existing account", zap.Error(err), zap.String("email", email.Value()))
		return nil, stackErr.Error(ErrCheckAccountFailed)
	}

	password, err := valueobject.NewPlainPassword(req.Password)
	if err != nil {
		log.Errorw("Failed to create password", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	hashedPassword, err := u.hasher.Hash(ctx, password.Value())
	if err != nil {
		log.Errorw("Failed to hash password", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	hashedPasswordVO, err := valueobject.NewHashedPassword(hashedPassword)
	if err != nil {
		log.Errorw("Failed to create hashed password value object", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	accountAggregate, err := aggregate.NewAccountAggregate(uuid.New().String())
	if err != nil {
		log.Errorw("Failed to create account aggregate", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	if err := accountAggregate.Register(email, hashedPasswordVO, req.DisplayName, now); err != nil {
		log.Errorw("Failed to register account aggregate", zap.Error(err))
		return nil, stackErr.Error(err)
	}
	newAccountEntity, err := accountAggregate.Snapshot()
	if err != nil {
		log.Errorw("Failed to build account projection", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.AccountAggregateRepository().Save(ctx, accountAggregate); err != nil {
			return fmt.Errorf("save account aggregate failed: %v", err)
		}
		return nil
	}); txErr != nil {
		log.Errorw("Failed to register account", zap.Error(txErr))
		return nil, stackErr.Error(txErr)
	}

	token, expiresAt, err := u.paseto.GenerateToken(ctx, newAccountEntity)
	if err != nil {
		log.Errorw("Failed to generate token", zap.Error(err))
		return nil, stackErr.Error(fmt.Errorf("generate token failed: %v", err))
	}

	return &out.RegisterResponse{
		Token:     token,
		ExpiresAt: expiresAt.UnixMilli(),
	}, nil
}
