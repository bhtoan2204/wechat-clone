package command

import (
	"context"
	"errors"

	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/application/dto/in"
	"go-socket/core/modules/account/application/dto/out"
	"go-socket/core/modules/account/application/service"
	repos "go-socket/core/modules/account/domain/repos"
	domainservice "go-socket/core/modules/account/domain/service"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type registerHandler struct {
	registrationService service.RegistrationService
}

func NewRegisterHandler(_ *appCtx.AppContext, _ repos.Repos, services service.Services) cqrs.Handler[*in.RegisterRequest, *out.RegisterResponse] {
	return &registerHandler{
		registrationService: services.RegistrationService(),
	}
}

func (u *registerHandler) Handle(ctx context.Context, req *in.RegisterRequest) (*out.RegisterResponse, error) {
	log := logging.FromContext(ctx).Named("Register")

	result, err := u.registrationService.Register(ctx, service.RegisterAccountCommand{
		Email:       req.Email,
		Password:    req.Password,
		DisplayName: req.DisplayName,
	})
	if err != nil {
		if errors.Is(err, service.ErrRegistrationAccountExists) || errors.Is(err, domainservice.ErrAccountEmailAlreadyExists) {
			log.Errorw("Account already exists", zap.String("email", req.Email))
			return nil, stackErr.Error(ErrAccountExists)
		}
		if errors.Is(err, service.ErrRegistrationCheckAccountFailed) {
			log.Errorw("Failed to check existing account", zap.Error(err), zap.String("email", req.Email))
			return nil, stackErr.Error(ErrCheckAccountFailed)
		}

		log.Errorw("Failed to register account", zap.Error(err), zap.String("email", req.Email))
		return nil, stackErr.Error(err)
	}

	return &out.RegisterResponse{
		AccessToken:      result.AccessToken,
		AccessExpiresAt:  result.AccessExpiresAt.UnixMilli(),
		RefreshToken:     result.RefreshToken,
		RefreshExpiresAt: result.RefreshExpiresAt.UnixMilli(),
	}, nil
}
