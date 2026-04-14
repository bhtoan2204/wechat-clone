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
	"gorm.io/gorm"
)

var (
	ErrAuthenticationAccountNotFound  = errors.New("authentication account not found")
	ErrAuthenticationInvalidPassword  = errors.New("authentication invalid password")
	ErrRegistrationAccountExists      = errors.New("registration account already exists")
	ErrRegistrationCheckAccountFailed = errors.New("registration check account failed")
	ErrRefreshTokenInvalid            = errors.New("refresh token is invalid")
	ErrRefreshSessionExpired          = errors.New("refresh session expired")
	ErrRefreshSessionRevoked          = errors.New("refresh session revoked")
)

type DeviceCommand struct {
	DeviceUID  string
	DeviceName string
	DeviceType string
	OSName     string
	OSVersion  string
	AppVersion string
	UserAgent  string
	IPAddress  string
}

type RegisterAccountCommand struct {
	Email       string
	Password    string
	DisplayName string
	Device      DeviceCommand
}

type AuthenticateAccountCommand struct {
	Email    string
	Password string
	Device   DeviceCommand
}

type TokenPairResult struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type RefreshTokenCommand struct {
	RefreshToken string
	UserAgent    string
	IPAddress    string
}

type LogoutCommand struct {
	AccountID    string
	RefreshToken string
}

type RevokeAccountSessionsCommand struct {
	AccountID string
	Reason    string
}

//go:generate mockgen -package=service -destination=authentication_service_mock.go -source=authentication_service.go
type AuthenticationService interface {
	Register(ctx context.Context, command RegisterAccountCommand) (*TokenPairResult, error)
	Authenticate(ctx context.Context, command AuthenticateAccountCommand) (*TokenPairResult, error)
	RefreshAuthenticate(ctx context.Context, command RefreshTokenCommand) (*TokenPairResult, error)
	Logout(ctx context.Context, command LogoutCommand) error
	RevokeAllSessions(ctx context.Context, command RevokeAccountSessionsCommand) error
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

func (s *authenticationService) Register(ctx context.Context, command RegisterAccountCommand) (*TokenPairResult, error) {
	now := time.Now().UTC()
	email, err := valueobject.NewEmail(command.Email)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	if err := domainservice.EnsureEmailAvailable(ctx, s.baseRepo.AccountRepository(), email); err != nil {
		if errors.Is(err, domainservice.ErrAccountEmailAlreadyExists) {
			return nil, stackErr.Error(ErrRegistrationAccountExists)
		}
		return nil, stackErr.Error(ErrRegistrationCheckAccountFailed)
	}

	password, err := valueobject.NewPlainPassword(command.Password)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	hashedPassword, err := s.hasher.Hash(ctx, password.Value())
	if err != nil {
		return nil, stackErr.Error(err)
	}

	hashedPasswordVO, err := valueobject.NewHashedPassword(hashedPassword)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accountAggregate, err := aggregate.NewAccountAggregate(uuid.NewString())
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := accountAggregate.Register(email, hashedPasswordVO, command.DisplayName, now); err != nil {
		return nil, stackErr.Error(err)
	}

	accountSnapshot, err := accountAggregate.Snapshot()
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var tokenPair *TokenPairResult
	if txErr := s.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		if err := txRepos.AccountAggregateRepository().Save(ctx, accountAggregate); err != nil {
			return stackErr.Error(fmt.Errorf("save account aggregate failed: %v", err))
		}
		device, err := s.ensureKnownDevice(ctx, txRepos.DeviceRepository(), accountSnapshot.ID, command.Device, now)
		if err != nil {
			return stackErr.Error(err)
		}
		tokenPair, err = s.createSessionTokenPair(ctx, txRepos.SessionRepository(), accountSnapshot, device.ID, command.Device, now)
		if err != nil {
			return stackErr.Error(err)
		}
		return nil
	}); txErr != nil {
		return nil, stackErr.Error(txErr)
	}

	return tokenPair, nil
}

func (s *authenticationService) Authenticate(ctx context.Context, command AuthenticateAccountCommand) (*TokenPairResult, error) {
	now := time.Now().UTC()
	email, err := valueobject.NewEmail(command.Email)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	password, err := valueobject.NewPlainPassword(command.Password)
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

	var tokenPair *TokenPairResult
	if txErr := s.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		device, err := s.ensureKnownDevice(ctx, txRepos.DeviceRepository(), accountSnapshot.ID, command.Device, now)
		if err != nil {
			return stackErr.Error(err)
		}
		tokenPair, err = s.createSessionTokenPair(ctx, txRepos.SessionRepository(), accountSnapshot, device.ID, command.Device, now)
		if err != nil {
			return stackErr.Error(err)
		}
		return nil
	}); txErr != nil {
		return nil, stackErr.Error(txErr)
	}

	return tokenPair, nil
}

func (s *authenticationService) RefreshAuthenticate(ctx context.Context, command RefreshTokenCommand) (*TokenPairResult, error) {
	claims, err := s.paseto.ParseRefreshToken(ctx, command.RefreshToken)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%w: %v", ErrRefreshTokenInvalid, err))
	}

	now := time.Now().UTC()
	var tokenPair *TokenPairResult
	if txErr := s.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		sessionAgg, err := txRepos.SessionRepository().Load(ctx, claims.SessionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return stackErr.Error(ErrRefreshTokenInvalid)
			}
			return stackErr.Error(fmt.Errorf("load session failed: %v", err))
		}
		session, err := sessionAgg.Snapshot()
		if err != nil {
			return stackErr.Error(err)
		}
		if session.AccountID != claims.AccountID || session.DeviceID != claims.DeviceID {
			return stackErr.Error(ErrRefreshTokenInvalid)
		}

		valid, err := s.hasher.Verify(ctx, command.RefreshToken, session.RefreshTokenHash)
		if err != nil {
			return stackErr.Error(fmt.Errorf("verify refresh token failed: %v", err))
		}
		if !valid {
			return stackErr.Error(ErrRefreshTokenInvalid)
		}

		if err := sessionAgg.EnsureRefreshAllowed(now); err != nil {
			if errors.Is(err, entity.ErrSessionExpired) && sessionAgg.MarkExpired(now) {
				if saveErr := txRepos.SessionRepository().Save(ctx, sessionAgg); saveErr != nil {
					return stackErr.Error(fmt.Errorf("mark session expired failed: %v", saveErr))
				}
			}
			return stackErr.Error(mapRefreshSessionErr(err))
		}

		accountAgg, err := txRepos.AccountAggregateRepository().Load(ctx, claims.AccountID)
		if err != nil {
			return stackErr.Error(fmt.Errorf("load account aggregate failed: %v", err))
		}
		accountSnapshot, err := accountAgg.Snapshot()
		if err != nil {
			return stackErr.Error(err)
		}

		device, err := txRepos.DeviceRepository().GetByAccountAndID(ctx, claims.AccountID, session.DeviceID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return stackErr.Error(ErrRefreshTokenInvalid)
			}
			return stackErr.Error(fmt.Errorf("load device failed: %v", err))
		}
		device.Touch(command.UserAgent, command.IPAddress, now)
		if err := txRepos.DeviceRepository().Save(ctx, device); err != nil {
			return stackErr.Error(fmt.Errorf("save device failed: %v", err))
		}

		tokenPair, err = issueTokenPair(ctx, s.paseto, accountSnapshot, xpaseto.RefreshTokenSubject{
			SessionID: sessionAgg.SessionID(),
			DeviceID:  sessionAgg.DeviceID(),
		})
		if err != nil {
			return stackErr.Error(err)
		}

		refreshTokenHash, err := s.hasher.Hash(ctx, tokenPair.RefreshToken)
		if err != nil {
			return stackErr.Error(fmt.Errorf("hash refresh token failed: %v", err))
		}
		if err := sessionAgg.Rotate(refreshTokenHash, tokenPair.RefreshExpiresAt, now, command.IPAddress, command.UserAgent); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.SessionRepository().Save(ctx, sessionAgg); err != nil {
			return stackErr.Error(fmt.Errorf("save rotated session failed: %v", err))
		}

		return nil
	}); txErr != nil {
		return nil, stackErr.Error(txErr)
	}

	return tokenPair, nil
}

func (s *authenticationService) Logout(ctx context.Context, command LogoutCommand) error {
	if command.AccountID == "" {
		return stackErr.Error(ErrRefreshTokenInvalid)
	}
	if command.RefreshToken == "" {
		return s.RevokeAllSessions(ctx, RevokeAccountSessionsCommand{
			AccountID: command.AccountID,
			Reason:    "logout_all",
		})
	}

	claims, err := s.paseto.ParseRefreshToken(ctx, command.RefreshToken)
	if err != nil {
		return stackErr.Error(err)
	}
	if claims.AccountID != command.AccountID {
		return stackErr.Error(ErrRefreshTokenInvalid)
	}

	now := time.Now().UTC()
	return stackErr.Error(s.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		sessionAgg, err := txRepos.SessionRepository().Load(ctx, claims.SessionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil
			}
			return stackErr.Error(err)
		}
		session, err := sessionAgg.Snapshot()
		if err != nil {
			return stackErr.Error(err)
		}
		if session.AccountID != command.AccountID || session.DeviceID != claims.DeviceID {
			return stackErr.Error(ErrRefreshTokenInvalid)
		}
		changed, err := sessionAgg.Revoke("logout", now)
		if err != nil {
			return stackErr.Error(err)
		}
		if !changed {
			return nil
		}
		if err := txRepos.SessionRepository().Save(ctx, sessionAgg); err != nil {
			return stackErr.Error(err)
		}
		return nil
	}))
}

func (s *authenticationService) RevokeAllSessions(ctx context.Context, command RevokeAccountSessionsCommand) error {
	if command.AccountID == "" {
		return stackErr.Error(ErrRefreshTokenInvalid)
	}

	now := time.Now().UTC()
	return stackErr.Error(s.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		sessionAggs, err := txRepos.SessionRepository().ListByAccountID(ctx, command.AccountID)
		if err != nil {
			return stackErr.Error(err)
		}
		for _, sessionAgg := range sessionAggs {
			changed, err := sessionAgg.Revoke(command.Reason, now)
			if err != nil {
				return stackErr.Error(err)
			}
			if !changed {
				continue
			}
			if err := txRepos.SessionRepository().Save(ctx, sessionAgg); err != nil {
				return stackErr.Error(err)
			}
		}
		return nil
	}))
}

func (s *authenticationService) ensureKnownDevice(
	ctx context.Context,
	deviceRepo repos.DeviceRepository,
	accountID string,
	command DeviceCommand,
	now time.Time,
) (*entity.Device, error) {
	registration := entity.DeviceRegistration{
		DeviceUID:  command.DeviceUID,
		DeviceName: command.DeviceName,
		DeviceType: command.DeviceType,
		OSName:     command.OSName,
		OSVersion:  command.OSVersion,
		AppVersion: command.AppVersion,
		UserAgent:  command.UserAgent,
		IPAddress:  command.IPAddress,
	}

	device, err := deviceRepo.FindByAccountAndUID(ctx, accountID, command.DeviceUID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, stackErr.Error(fmt.Errorf("load device failed: %v", err))
		}
		device, err = entity.NewDevice(uuid.NewString(), accountID, registration, now)
		if err != nil {
			return nil, stackErr.Error(err)
		}
	} else {
		if err := device.RefreshRegistration(registration, now); err != nil {
			return nil, stackErr.Error(err)
		}
	}

	if err := deviceRepo.Save(ctx, device); err != nil {
		return nil, stackErr.Error(fmt.Errorf("save device failed: %v", err))
	}
	return device, nil
}

func (s *authenticationService) createSessionTokenPair(
	ctx context.Context,
	sessionRepo repos.SessionRepository,
	account *entity.Account,
	deviceID string,
	command DeviceCommand,
	now time.Time,
) (*TokenPairResult, error) {
	sessionID := uuid.NewString()
	tokenPair, err := issueTokenPair(ctx, s.paseto, account, xpaseto.RefreshTokenSubject{
		SessionID: sessionID,
		DeviceID:  deviceID,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	refreshTokenHash, err := s.hasher.Hash(ctx, tokenPair.RefreshToken)
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("hash refresh token failed: %v", err))
	}

	sessionAgg, err := aggregate.NewSessionAggregate(sessionID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := sessionAgg.Create(
		account.ID,
		deviceID,
		refreshTokenHash,
		tokenPair.RefreshExpiresAt,
		now,
		command.IPAddress,
		command.UserAgent,
	); err != nil {
		return nil, stackErr.Error(err)
	}

	if err := sessionRepo.Save(ctx, sessionAgg); err != nil {
		return nil, stackErr.Error(fmt.Errorf("save session failed: %v", err))
	}
	return tokenPair, nil
}

func mapRefreshSessionErr(err error) error {
	switch {
	case errors.Is(err, entity.ErrSessionExpired):
		return ErrRefreshSessionExpired
	case errors.Is(err, entity.ErrSessionRevoked):
		return ErrRefreshSessionRevoked
	default:
		return ErrRefreshTokenInvalid
	}
}
