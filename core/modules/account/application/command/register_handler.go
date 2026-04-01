package command

import (
	"context"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/domain/aggregate"
	"go-socket/core/modules/account/domain/entity"
	repos "go-socket/core/modules/account/domain/repos"
	valueobject "go-socket/core/modules/account/domain/value_object"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/cqrs"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/hasher"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"reflect"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
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
	log := logging.FromContext(ctx).Named("Register")
	accountRepo := u.baseRepo.AccountRepository()
	exists, err := accountRepo.IsEmailExists(ctx, req.Email)
	if err != nil {
		log.Errorw("Failed to check existing account", zap.Error(err))
		return nil, stackerr.Error(ErrCheckAccountFailed)
	}
	if exists {
		log.Errorw("Account already exists", zap.String("email", req.Email))
		return nil, stackerr.Error(ErrAccountExists)
	}

	password, err := valueobject.NewPassword(req.Password)
	if err != nil {
		log.Errorw("Failed to create password", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	hashedPassword, err := u.hasher.Hash(ctx, password.Value())
	if err != nil {
		log.Errorw("Failed to hash password", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	email, err := valueobject.NewEmail(req.Email)
	if err != nil {
		log.Errorw("Failed to create email", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	hashedPasswordVO, err := valueobject.NewPassword(hashedPassword)
	if err != nil {
		log.Errorw("Failed to create hashed password value object", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	newAccountEntity := &entity.Account{
		ID:       uuid.New().String(),
		Email:    email,
		Password: hashedPasswordVO,
	}

	if txErr := u.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.AccountRepository().CreateAccount(ctx, newAccountEntity); err != nil {
			log.Errorw("Failed to create account", zap.Error(err))
			return stackerr.Error(fmt.Errorf("create account failed: %w", err))
		}

		accountAggregate := &aggregate.AccountAggregate{}
		accountAggregateType := reflect.TypeOf(accountAggregate).Elem().Name()
		accountAggregate.SetAggregateType(accountAggregateType)
		if err := accountAggregate.SetID(newAccountEntity.ID); err != nil {
			return fmt.Errorf("set account aggregate id failed: %w", err)
		}

		if err := accountAggregate.ApplyChange(accountAggregate, &aggregate.EventAccountCreated{
			AccountID: newAccountEntity.ID,
			Email:     newAccountEntity.Email.Value(),
			CreatedAt: time.Now(),
		}); err != nil {
			return fmt.Errorf("apply account created event failed: %w", err)
		}

		publisher := eventpkg.NewPublisher(txRepos.AccountOutboxEventsRepository())
		if err := publisher.PublishAggregate(ctx, accountAggregate); err != nil {
			return fmt.Errorf("publish account created event failed: %w", err)
		}
		return nil
	}); txErr != nil {
		log.Errorw("Failed to register account", zap.Error(txErr))
		return nil, stackerr.Error(txErr)
	}

	token, expiresAt, err := u.paseto.GenerateToken(ctx, newAccountEntity)
	if err != nil {
		log.Errorw("Failed to generate token", zap.Error(err))
		return nil, stackerr.Error(fmt.Errorf("generate token failed: %w", err))
	}

	return &out.RegisterResponse{
		Token:     token,
		ExpiresAt: expiresAt.UnixMilli(),
	}, nil
}
