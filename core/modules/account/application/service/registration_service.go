package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/domain/aggregate"
	"go-socket/core/modules/account/domain/entity"
	repos "go-socket/core/modules/account/domain/repos"
	domainservice "go-socket/core/modules/account/domain/service"
	valueobject "go-socket/core/modules/account/domain/value_object"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/hasher"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

var (
	ErrRegistrationAccountExists      = errors.New("registration account already exists")
	ErrRegistrationCheckAccountFailed = errors.New("registration check account failed")
)

type RegisterAccountCommand struct {
	Email       string
	Password    string
	DisplayName string
}

type RegistrationResult struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

//go:generate mockgen -package=service -destination=registration_service_mock.go -source=registration_service.go
type RegistrationService interface {
	Register(ctx context.Context, command RegisterAccountCommand) (*RegistrationResult, error)
}

type registrationService struct {
	baseRepo repos.Repos
	hasher   hasher.Hasher
	paseto   xpaseto.PasetoService
}

type registerAggregateParams struct {
	AccountID    string
	Email        valueobject.Email
	PasswordHash valueobject.HashedPassword
	DisplayName  string
	RegisteredAt time.Time
}

func NewRegistrationService(appCtx *appCtx.AppContext, baseRepo repos.Repos) RegistrationService {
	return &registrationService{
		baseRepo: baseRepo,
		hasher:   appCtx.GetHasher(),
		paseto:   appCtx.GetPaseto(),
	}
}

func (s *registrationService) Register(ctx context.Context, command RegisterAccountCommand) (*RegistrationResult, error) {
	email, hashedPassword, err := s.prepareRegistrationCredentials(ctx, command)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountAggregate, accountSnapshot, err := s.buildRegisteredAggregate(registerAggregateParams{
		AccountID:    uuid.NewString(),
		Email:        email,
		PasswordHash: hashedPassword,
		DisplayName:  command.DisplayName,
		RegisteredAt: time.Now().UTC(),
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := s.persistRegisteredAggregate(ctx, accountAggregate); err != nil {
		return nil, stackErr.Error(err)
	}

	result, err := s.issueRegistrationSession(ctx, accountSnapshot)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return result, nil
}

func (s *registrationService) prepareRegistrationCredentials(ctx context.Context, command RegisterAccountCommand) (valueobject.Email, valueobject.HashedPassword, error) {
	email, err := valueobject.NewEmail(command.Email)
	if err != nil {
		return valueobject.Email{}, valueobject.HashedPassword{}, stackErr.Error(err)
	}

	if err := s.ensureEmailAvailable(ctx, email); err != nil {
		return valueobject.Email{}, valueobject.HashedPassword{}, stackErr.Error(err)
	}

	password, err := valueobject.NewPlainPassword(command.Password)
	if err != nil {
		return valueobject.Email{}, valueobject.HashedPassword{}, stackErr.Error(err)
	}

	hashedPassword, err := s.hasher.Hash(ctx, password.Value())
	if err != nil {
		return valueobject.Email{}, valueobject.HashedPassword{}, stackErr.Error(err)
	}

	hashedPasswordVO, err := valueobject.NewHashedPassword(hashedPassword)
	if err != nil {
		return valueobject.Email{}, valueobject.HashedPassword{}, stackErr.Error(err)
	}

	return email, hashedPasswordVO, nil
}

func (s *registrationService) ensureEmailAvailable(ctx context.Context, email valueobject.Email) error {
	if err := domainservice.EnsureEmailAvailable(ctx, s.baseRepo.AccountRepository(), email); err != nil {
		if errors.Is(err, domainservice.ErrAccountEmailAlreadyExists) {
			return stackErr.Error(ErrRegistrationAccountExists)
		}
		return stackErr.Error(ErrRegistrationCheckAccountFailed)
	}
	return nil
}

func (s *registrationService) buildRegisteredAggregate(params registerAggregateParams) (*aggregate.AccountAggregate, *entity.Account, error) {
	accountAggregate, err := aggregate.NewAccountAggregate(params.AccountID)
	if err != nil {
		return nil, nil, stackErr.Error(err)
	}
	if err := accountAggregate.Register(params.Email, params.PasswordHash, params.DisplayName, params.RegisteredAt); err != nil {
		return nil, nil, stackErr.Error(err)
	}

	accountSnapshot, err := accountAggregate.Snapshot()
	if err != nil {
		return nil, nil, stackErr.Error(err)
	}
	return accountAggregate, accountSnapshot, nil
}

func (s *registrationService) persistRegisteredAggregate(ctx context.Context, accountAggregate *aggregate.AccountAggregate) error {
	if txErr := s.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.AccountAggregateRepository().Save(ctx, accountAggregate); err != nil {
			return stackErr.Error(fmt.Errorf("save account aggregate failed: %v", err))
		}
		return nil
	}); txErr != nil {
		return stackErr.Error(txErr)
	}
	return nil
}

func (s *registrationService) issueRegistrationSession(ctx context.Context, accountSnapshot *entity.Account) (*RegistrationResult, error) {
	accessToken, accessExpiresAt, err := s.paseto.GenerateAccessToken(ctx, accountSnapshot)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("generate token failed: %v", err))
	}

	refreshToken, refrestExpiresAt, err := s.paseto.GenerateAccessToken(ctx, accountSnapshot)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("generate token failed: %v", err))
	}

	return &RegistrationResult{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExpiresAt,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refrestExpiresAt,
	}, nil
}
