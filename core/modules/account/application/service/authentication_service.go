package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	appCtx "go-socket/core/context"
	repos "go-socket/core/modules/account/domain/repos"
	valueobject "go-socket/core/modules/account/domain/value_object"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/hasher"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

var (
	ErrAuthenticationAccountNotFound = errors.New("authentication account not found")
	ErrAuthenticationInvalidPassword = errors.New("authentication invalid password")
)

type AuthenticateAccountCommand struct {
	Email    string
	Password string
}

type AuthenticationResult struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

//go:generate mockgen -package=service -destination=authentication_service_mock.go -source=authentication_service.go
type AuthenticationService interface {
	Authenticate(ctx context.Context, command AuthenticateAccountCommand) (*AuthenticationResult, error)
}

type authenticationService struct {
	baseRepo repos.Repos
	hasher   hasher.Hasher
	paseto   xpaseto.PasetoService
}

func NewAuthenticationService(appCtx *appCtx.AppContext, baseRepo repos.Repos) AuthenticationService {
	return &authenticationService{
		baseRepo: baseRepo,
		hasher:   appCtx.GetHasher(),
		paseto:   appCtx.GetPaseto(),
	}
}

func (s *authenticationService) Authenticate(ctx context.Context, command AuthenticateAccountCommand) (*AuthenticationResult, error) {
	email, password, err := s.prepareCredentials(command)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountAggregate, err := s.baseRepo.AccountAggregateRepository().LoadByEmail(ctx, email.Value())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, stackErr.Error(ErrAuthenticationAccountNotFound)
		}
		return nil, stackErr.Error(fmt.Errorf("load account aggregate by email failed: %v", err))
	}

	currentHash, err := accountAggregate.CurrentPasswordHash()
	if err != nil {
		return nil, stackErr.Error(err)
	}

	valid, err := s.hasher.Verify(ctx, password.Value(), currentHash.Value())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if !valid {
		return nil, stackErr.Error(ErrAuthenticationInvalidPassword)
	}

	accountSnapshot, err := accountAggregate.Snapshot()
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accessToken, accessExpiresAt, err := s.paseto.GenerateAccessToken(ctx, accountSnapshot)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("generate token failed: %v", err))
	}

	refreshToken, refrestExpiresAt, err := s.paseto.GenerateAccessToken(ctx, accountSnapshot)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("generate token failed: %v", err))
	}

	return &AuthenticationResult{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refrestExpiresAt,
	}, nil
}

func (s *authenticationService) prepareCredentials(command AuthenticateAccountCommand) (valueobject.Email, valueobject.PlainPassword, error) {
	email, err := valueobject.NewEmail(command.Email)
	if err != nil {
		return valueobject.Email{}, valueobject.PlainPassword{}, stackErr.Error(err)
	}

	password, err := valueobject.NewPlainPassword(command.Password)
	if err != nil {
		return valueobject.Email{}, valueobject.PlainPassword{}, stackErr.Error(err)
	}

	return email, password, nil
}
