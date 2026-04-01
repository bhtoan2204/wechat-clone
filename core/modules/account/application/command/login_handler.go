package command

import (
	"context"
	"errors"
	"fmt"
	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	repos "go-socket/core/modules/account/domain/repos"
	valueobject "go-socket/core/modules/account/domain/value_object"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/hasher"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type loginHandler struct {
	accountRepo repos.AccountRepository
	hasher      hasher.Hasher
	paseto      xpaseto.PasetoService
}

func NewLoginHandler(appCtx *appCtx.AppContext, baseRepo repos.Repos) cqrs.Handler[*in.LoginRequest, *out.LoginResponse] {
	return &loginHandler{
		accountRepo: baseRepo.AccountRepository(),
		hasher:      appCtx.GetHasher(),
		paseto:      appCtx.GetPaseto(),
	}
}

func (u *loginHandler) Handle(ctx context.Context, req *in.LoginRequest) (*out.LoginResponse, error) {
	log := logging.FromContext(ctx).Named("Login")
	email, err := valueobject.NewEmail(req.Email)
	if err != nil {
		log.Errorw("Invalid email", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	password, err := valueobject.NewPassword(req.Password)
	if err != nil {
		log.Errorw("Invalid password", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	account, err := u.accountRepo.GetAccountByEmail(ctx, email.Value())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Errorw("Account not found", zap.String("email", email.Value()))
			return nil, stackerr.Error(ErrAccountNotFound)
		}
		log.Errorw("Failed to get account", zap.Error(err))
		return nil, stackerr.Error(fmt.Errorf("get account failed: %w", err))
	}

	valid, err := u.hasher.Verify(ctx, password.Value(), account.Password.Value())
	if err != nil {
		log.Errorw("Failed to verify password", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if !valid {
		log.Errorw("Invalid credentials", zap.String("email", email.Value()))
		return nil, stackerr.Error(ErrInvalidCredentials)
	}

	token, expiresAt, err := u.paseto.GenerateToken(ctx, account)
	if err != nil {
		log.Errorw("Failed to generate token", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	return &out.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt.UnixMilli(),
	}, nil
}
