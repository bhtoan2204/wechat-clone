package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-socket/core/modules/account/domain/aggregate"
	"go-socket/core/modules/account/domain/entity"
	"go-socket/core/modules/account/domain/repos"
	valueobject "go-socket/core/modules/account/domain/value_object"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/hasher"

	"go.uber.org/mock/gomock"
	"gorm.io/gorm"
)

func TestAuthenticationService_Authenticate_IssuesAccessAndRefreshTokens(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	accountAggregate := newRegisteredAccountAggregate(t, "acc-1", "alice@example.com", "hashed-password")
	accountAggregateRepo := repos.NewMockAccountAggregateRepository(ctrl)
	deviceRepo := repos.NewMockDeviceRepository(ctrl)
	sessionRepo := repos.NewMockSessionRepository(ctrl)
	baseRepo := repos.NewMockRepos(ctrl)
	txRepos := repos.NewMockRepos(ctrl)
	hasherMock := hasher.NewMockHasher(ctrl)
	pasetoMock := xpaseto.NewMockPasetoService(ctrl)

	accessExpiresAt := time.Date(2026, time.April, 14, 11, 0, 0, 0, time.UTC)
	refreshExpiresAt := time.Date(2026, time.April, 21, 11, 0, 0, 0, time.UTC)

	var savedDeviceID string
	var issuedSessionID string

	baseRepo.EXPECT().AccountAggregateRepository().Return(accountAggregateRepo)
	accountAggregateRepo.EXPECT().LoadByEmail(gomock.Any(), "alice@example.com").Return(accountAggregate, nil)
	hasherMock.EXPECT().Verify(gomock.Any(), "password123", "hashed-password").Return(true, nil)
	baseRepo.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(repos.Repos) error) error {
			return fn(txRepos)
		})
	txRepos.EXPECT().DeviceRepository().Return(deviceRepo)
	deviceRepo.EXPECT().FindByAccountAndUID(gomock.Any(), "acc-1", "browser-1").Return(nil, gorm.ErrRecordNotFound)
	deviceRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.DeviceAggregate{})).
		DoAndReturn(func(_ context.Context, deviceAgg *aggregate.DeviceAggregate) error {
			device, err := deviceAgg.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if device == nil || device.AccountID != "acc-1" || device.DeviceUID != "browser-1" {
				t.Fatalf("expected device for acc-1/browser-1, got %+v", device)
			}
			savedDeviceID = device.ID
			return nil
		})
	pasetoMock.EXPECT().
		GenerateAccessToken(gomock.Any(), gomock.AssignableToTypeOf(&entity.Account{})).
		DoAndReturn(func(_ context.Context, account *entity.Account) (string, time.Time, error) {
			if account == nil || account.ID != "acc-1" {
				t.Fatalf("expected account snapshot from aggregate, got %+v", account)
			}
			return "access-token", accessExpiresAt, nil
		})
	pasetoMock.EXPECT().
		GenerateRefreshToken(
			gomock.Any(),
			gomock.AssignableToTypeOf(&entity.Account{}),
			gomock.AssignableToTypeOf(xpaseto.RefreshTokenSubject{}),
		).
		DoAndReturn(func(_ context.Context, account *entity.Account, subject xpaseto.RefreshTokenSubject) (string, time.Time, error) {
			if account == nil || account.ID != "acc-1" {
				t.Fatalf("expected account snapshot from aggregate, got %+v", account)
			}
			if subject.DeviceID == "" || subject.DeviceID != savedDeviceID {
				t.Fatalf("expected refresh token subject bound to saved device, got %+v", subject)
			}
			if subject.SessionID == "" {
				t.Fatalf("expected non-empty session id in refresh token subject")
			}
			issuedSessionID = subject.SessionID
			return "refresh-token", refreshExpiresAt, nil
		})
	hasherMock.EXPECT().Hash(gomock.Any(), "refresh-token").Return("hashed-refresh-token", nil)
	txRepos.EXPECT().SessionRepository().Return(sessionRepo)
	sessionRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.SessionAggregate{})).
		DoAndReturn(func(_ context.Context, sessionAgg *aggregate.SessionAggregate) error {
			session, err := sessionAgg.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if session == nil || session.AccountID != "acc-1" || session.DeviceID != savedDeviceID {
				t.Fatalf("expected session for acc-1/%s, got %+v", savedDeviceID, session)
			}
			if session.ID != issuedSessionID {
				t.Fatalf("expected session id %q, got %q", issuedSessionID, session.ID)
			}
			if session.RefreshTokenHash != "hashed-refresh-token" {
				t.Fatalf("expected hashed refresh token, got %q", session.RefreshTokenHash)
			}
			return nil
		})

	service := &authenticationService{
		baseRepo: baseRepo,
		hasher:   hasherMock,
		paseto:   pasetoMock,
	}

	result, err := service.Authenticate(context.Background(), AuthenticateAccountCommand{
		Email:    "alice@example.com",
		Password: "password123",
		Device: DeviceCommand{
			DeviceUID: "browser-1",
			UserAgent: "Mozilla/5.0",
			IPAddress: "203.0.113.10",
		},
	})
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if result.AccessToken != "access-token" {
		t.Fatalf("expected access-token, got %q", result.AccessToken)
	}
	if result.RefreshToken != "refresh-token" {
		t.Fatalf("expected refresh-token, got %q", result.RefreshToken)
	}
	if !result.AccessExpiresAt.Equal(accessExpiresAt) {
		t.Fatalf("expected access expiry %v, got %v", accessExpiresAt, result.AccessExpiresAt)
	}
	if !result.RefreshExpiresAt.Equal(refreshExpiresAt) {
		t.Fatalf("expected refresh expiry %v, got %v", refreshExpiresAt, result.RefreshExpiresAt)
	}
}

func TestAuthenticationService_RefreshAuthenticate_RotatesRefreshToken(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	accountAggregate := newRegisteredAccountAggregate(t, "acc-1", "alice@example.com", "hashed-password")
	accountAggregateRepo := repos.NewMockAccountAggregateRepository(ctrl)
	deviceRepo := repos.NewMockDeviceRepository(ctrl)
	sessionRepo := repos.NewMockSessionRepository(ctrl)
	baseRepo := repos.NewMockRepos(ctrl)
	txRepos := repos.NewMockRepos(ctrl)
	hasherMock := hasher.NewMockHasher(ctrl)
	pasetoMock := xpaseto.NewMockPasetoService(ctrl)

	accessExpiresAt := time.Date(2026, time.April, 14, 12, 0, 0, 0, time.UTC)
	refreshExpiresAt := time.Date(2026, time.April, 21, 12, 0, 0, 0, time.UTC)
	deviceAgg := newKnownDeviceAggregate(t, "acc-1", "dev-1", "browser-1")
	sessionAgg := newActiveSessionAggregate(t, "ses-1", "acc-1", "dev-1", "stored-refresh-hash", refreshExpiresAt)

	pasetoMock.EXPECT().
		ParseRefreshToken(gomock.Any(), "incoming-refresh-token").
		Return(&xpaseto.PasetoPayload{
			AccountID: "acc-1",
			SessionID: "ses-1",
			DeviceID:  "dev-1",
			TokenType: xpaseto.TokenTypeRefresh,
		}, nil)
	baseRepo.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(repos.Repos) error) error {
			return fn(txRepos)
		})
	txRepos.EXPECT().SessionRepository().Return(sessionRepo).Times(2)
	sessionRepo.EXPECT().Load(gomock.Any(), "ses-1").Return(sessionAgg, nil)
	hasherMock.EXPECT().Verify(gomock.Any(), "incoming-refresh-token", "stored-refresh-hash").Return(true, nil)
	txRepos.EXPECT().AccountAggregateRepository().Return(accountAggregateRepo)
	accountAggregateRepo.EXPECT().Load(gomock.Any(), "acc-1").Return(accountAggregate, nil)
	txRepos.EXPECT().DeviceRepository().Return(deviceRepo).Times(2)
	deviceRepo.EXPECT().GetByAccountAndID(gomock.Any(), "acc-1", "dev-1").Return(deviceAgg, nil)
	deviceRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.DeviceAggregate{})).
		DoAndReturn(func(_ context.Context, savedAgg *aggregate.DeviceAggregate) error {
			saved, err := savedAgg.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if saved == nil || saved.ID != "dev-1" {
				t.Fatalf("expected saved device dev-1, got %+v", saved)
			}
			return nil
		})
	pasetoMock.EXPECT().
		GenerateAccessToken(gomock.Any(), gomock.AssignableToTypeOf(&entity.Account{})).
		Return("rotated-access-token", accessExpiresAt, nil)
	pasetoMock.EXPECT().
		GenerateRefreshToken(
			gomock.Any(),
			gomock.AssignableToTypeOf(&entity.Account{}),
			xpaseto.RefreshTokenSubject{SessionID: "ses-1", DeviceID: "dev-1"},
		).
		Return("rotated-refresh-token", refreshExpiresAt, nil)
	hasherMock.EXPECT().Hash(gomock.Any(), "rotated-refresh-token").Return("rotated-refresh-hash", nil)
	sessionRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.SessionAggregate{})).
		DoAndReturn(func(_ context.Context, sessionAgg *aggregate.SessionAggregate) error {
			session, err := sessionAgg.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if session == nil || session.ID != "ses-1" {
				t.Fatalf("expected rotated session ses-1, got %+v", session)
			}
			if session.RefreshTokenHash != "rotated-refresh-hash" {
				t.Fatalf("expected rotated hash, got %q", session.RefreshTokenHash)
			}
			return nil
		})

	service := &authenticationService{
		baseRepo: baseRepo,
		hasher:   hasherMock,
		paseto:   pasetoMock,
	}

	result, err := service.RefreshAuthenticate(context.Background(), RefreshTokenCommand{
		RefreshToken: "incoming-refresh-token",
		UserAgent:    "Mozilla/5.0",
		IPAddress:    "203.0.113.11",
	})
	if err != nil {
		t.Fatalf("RefreshAuthenticate() error = %v", err)
	}

	if result.AccessToken != "rotated-access-token" {
		t.Fatalf("expected rotated-access-token, got %q", result.AccessToken)
	}
	if result.RefreshToken != "rotated-refresh-token" {
		t.Fatalf("expected rotated-refresh-token, got %q", result.RefreshToken)
	}
	if !result.AccessExpiresAt.Equal(accessExpiresAt) {
		t.Fatalf("expected access expiry %v, got %v", accessExpiresAt, result.AccessExpiresAt)
	}
	if !result.RefreshExpiresAt.Equal(refreshExpiresAt) {
		t.Fatalf("expected refresh expiry %v, got %v", refreshExpiresAt, result.RefreshExpiresAt)
	}
}

func TestAuthenticationService_Authenticate_MapsNotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	accountAggregateRepo := repos.NewMockAccountAggregateRepository(ctrl)
	baseRepo := repos.NewMockRepos(ctrl)
	hasherMock := hasher.NewMockHasher(ctrl)
	pasetoMock := xpaseto.NewMockPasetoService(ctrl)

	baseRepo.EXPECT().AccountAggregateRepository().Return(accountAggregateRepo)
	accountAggregateRepo.EXPECT().LoadByEmail(gomock.Any(), "missing@example.com").Return(nil, gorm.ErrRecordNotFound)

	service := &authenticationService{
		baseRepo: baseRepo,
		hasher:   hasherMock,
		paseto:   pasetoMock,
	}

	_, err := service.Authenticate(context.Background(), AuthenticateAccountCommand{
		Email:    "missing@example.com",
		Password: "password123",
		Device: DeviceCommand{
			DeviceUID: "browser-1",
		},
	})

	if !errors.Is(err, ErrAuthenticationAccountNotFound) {
		t.Fatalf("expected ErrAuthenticationAccountNotFound, got %v", err)
	}
}

func TestAuthenticationService_Authenticate_MapsInvalidPassword(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	accountAggregate := newRegisteredAccountAggregate(t, "acc-1", "alice@example.com", "hashed-password")
	accountAggregateRepo := repos.NewMockAccountAggregateRepository(ctrl)
	baseRepo := repos.NewMockRepos(ctrl)
	hasherMock := hasher.NewMockHasher(ctrl)
	pasetoMock := xpaseto.NewMockPasetoService(ctrl)

	baseRepo.EXPECT().AccountAggregateRepository().Return(accountAggregateRepo)
	accountAggregateRepo.EXPECT().LoadByEmail(gomock.Any(), "alice@example.com").Return(accountAggregate, nil)
	hasherMock.EXPECT().Verify(gomock.Any(), "password123", "hashed-password").Return(false, nil)

	service := &authenticationService{
		baseRepo: baseRepo,
		hasher:   hasherMock,
		paseto:   pasetoMock,
	}

	_, err := service.Authenticate(context.Background(), AuthenticateAccountCommand{
		Email:    "alice@example.com",
		Password: "password123",
		Device: DeviceCommand{
			DeviceUID: "browser-1",
		},
	})

	if !errors.Is(err, ErrAuthenticationInvalidPassword) {
		t.Fatalf("expected ErrAuthenticationInvalidPassword, got %v", err)
	}
}

func TestAuthenticationService_Register_IssuesAccessAndRefreshTokens(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	baseRepo := repos.NewMockRepos(ctrl)
	txRepos := repos.NewMockRepos(ctrl)
	accountRepo := repos.NewMockAccountRepository(ctrl)
	accountAggregateRepo := repos.NewMockAccountAggregateRepository(ctrl)
	deviceRepo := repos.NewMockDeviceRepository(ctrl)
	sessionRepo := repos.NewMockSessionRepository(ctrl)
	hasherMock := hasher.NewMockHasher(ctrl)
	pasetoMock := xpaseto.NewMockPasetoService(ctrl)

	accessExpiresAt := time.Date(2026, time.April, 14, 10, 0, 0, 0, time.UTC)
	refreshExpiresAt := time.Date(2026, time.April, 21, 10, 0, 0, 0, time.UTC)

	var savedDeviceID string
	var issuedSessionID string

	baseRepo.EXPECT().AccountRepository().Return(accountRepo)
	accountRepo.EXPECT().IsEmailExists(gomock.Any(), "alice@example.com").Return(false, nil)
	hasherMock.EXPECT().Hash(gomock.Any(), "password123").Return("hashed-password", nil)
	baseRepo.EXPECT().
		WithTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(repos.Repos) error) error {
			return fn(txRepos)
		})
	txRepos.EXPECT().AccountAggregateRepository().Return(accountAggregateRepo)
	accountAggregateRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.AccountAggregate{})).
		Return(nil)
	txRepos.EXPECT().DeviceRepository().Return(deviceRepo)
	deviceRepo.EXPECT().FindByAccountAndUID(gomock.Any(), gomock.Any(), "browser-1").Return(nil, gorm.ErrRecordNotFound)
	deviceRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.DeviceAggregate{})).
		DoAndReturn(func(_ context.Context, deviceAgg *aggregate.DeviceAggregate) error {
			device, err := deviceAgg.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if device == nil || device.AccountID == "" || device.DeviceUID != "browser-1" {
				t.Fatalf("expected saved register device, got %+v", device)
			}
			savedDeviceID = device.ID
			return nil
		})
	pasetoMock.EXPECT().
		GenerateAccessToken(gomock.Any(), gomock.AssignableToTypeOf(&entity.Account{})).
		DoAndReturn(func(_ context.Context, account *entity.Account) (string, time.Time, error) {
			if account == nil || account.ID == "" {
				t.Fatalf("expected non-nil account snapshot, got %+v", account)
			}
			return "access-token", accessExpiresAt, nil
		})
	pasetoMock.EXPECT().
		GenerateRefreshToken(
			gomock.Any(),
			gomock.AssignableToTypeOf(&entity.Account{}),
			gomock.AssignableToTypeOf(xpaseto.RefreshTokenSubject{}),
		).
		DoAndReturn(func(_ context.Context, account *entity.Account, subject xpaseto.RefreshTokenSubject) (string, time.Time, error) {
			if account == nil || account.ID == "" {
				t.Fatalf("expected non-nil account snapshot, got %+v", account)
			}
			if subject.DeviceID == "" || subject.DeviceID != savedDeviceID {
				t.Fatalf("expected refresh token subject bound to saved device, got %+v", subject)
			}
			if subject.SessionID == "" {
				t.Fatalf("expected non-empty session id")
			}
			issuedSessionID = subject.SessionID
			return "refresh-token", refreshExpiresAt, nil
		})
	hasherMock.EXPECT().Hash(gomock.Any(), "refresh-token").Return("hashed-refresh-token", nil)
	txRepos.EXPECT().SessionRepository().Return(sessionRepo)
	sessionRepo.EXPECT().
		Save(gomock.Any(), gomock.AssignableToTypeOf(&aggregate.SessionAggregate{})).
		DoAndReturn(func(_ context.Context, sessionAgg *aggregate.SessionAggregate) error {
			session, err := sessionAgg.Snapshot()
			if err != nil {
				t.Fatalf("Snapshot() error = %v", err)
			}
			if session == nil || session.ID != issuedSessionID || session.DeviceID != savedDeviceID {
				t.Fatalf("expected register session bound to issued subject, got %+v", session)
			}
			return nil
		})

	service := &authenticationService{
		baseRepo: baseRepo,
		hasher:   hasherMock,
		paseto:   pasetoMock,
	}

	result, err := service.Register(context.Background(), RegisterAccountCommand{
		Email:       "alice@example.com",
		Password:    "password123",
		DisplayName: "Alice",
		Device: DeviceCommand{
			DeviceUID: "browser-1",
			UserAgent: "Mozilla/5.0",
			IPAddress: "203.0.113.12",
		},
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if result.AccessToken != "access-token" {
		t.Fatalf("expected access-token, got %q", result.AccessToken)
	}
	if result.RefreshToken != "refresh-token" {
		t.Fatalf("expected refresh-token, got %q", result.RefreshToken)
	}
	if !result.AccessExpiresAt.Equal(accessExpiresAt) {
		t.Fatalf("expected access expiry %v, got %v", accessExpiresAt, result.AccessExpiresAt)
	}
	if !result.RefreshExpiresAt.Equal(refreshExpiresAt) {
		t.Fatalf("expected refresh expiry %v, got %v", refreshExpiresAt, result.RefreshExpiresAt)
	}
}

func TestAuthenticationService_Register_ReturnsAccountExists(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	baseRepo := repos.NewMockRepos(ctrl)
	accountRepo := repos.NewMockAccountRepository(ctrl)
	hasherMock := hasher.NewMockHasher(ctrl)
	pasetoMock := xpaseto.NewMockPasetoService(ctrl)

	baseRepo.EXPECT().AccountRepository().Return(accountRepo)
	accountRepo.EXPECT().IsEmailExists(gomock.Any(), "alice@example.com").Return(true, nil)

	service := &authenticationService{
		baseRepo: baseRepo,
		hasher:   hasherMock,
		paseto:   pasetoMock,
	}

	_, err := service.Register(context.Background(), RegisterAccountCommand{
		Email:       "alice@example.com",
		Password:    "password123",
		DisplayName: "Alice",
		Device: DeviceCommand{
			DeviceUID: "browser-1",
		},
	})
	if !errors.Is(err, ErrRegistrationAccountExists) {
		t.Fatalf("expected ErrRegistrationAccountExists, got %v", err)
	}
}

func newRegisteredAccountAggregate(t *testing.T, accountID, emailValue, passwordHashValue string) *aggregate.AccountAggregate {
	t.Helper()

	email, err := valueobject.NewEmail(emailValue)
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}

	passwordHash, err := valueobject.NewHashedPassword(passwordHashValue)
	if err != nil {
		t.Fatalf("NewHashedPassword() error = %v", err)
	}

	accountAggregate, err := aggregate.NewAccountAggregate(accountID)
	if err != nil {
		t.Fatalf("NewAccountAggregate() error = %v", err)
	}

	now := time.Date(2026, time.April, 14, 8, 0, 0, 0, time.UTC)
	if err := accountAggregate.Register(email, passwordHash, "Alice", now); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	return accountAggregate
}

func newActiveSession(sessionID, accountID, deviceID, refreshHash string, expiresAt time.Time) *entity.Session {
	now := time.Date(2026, time.April, 14, 9, 0, 0, 0, time.UTC)
	return &entity.Session{
		ID:               sessionID,
		AccountID:        accountID,
		DeviceID:         deviceID,
		RefreshTokenHash: refreshHash,
		Status:           entity.SessionStatusActive,
		LastActivityAt:   &now,
		ExpiresAt:        expiresAt,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
}

func newKnownDevice(accountID, deviceID, deviceUID string) *entity.Device {
	now := time.Date(2026, time.April, 14, 9, 0, 0, 0, time.UTC)
	return &entity.Device{
		ID:         deviceID,
		AccountID:  accountID,
		DeviceUID:  deviceUID,
		DeviceType: entity.DeviceTypeWeb,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func newActiveSessionAggregate(
	t *testing.T,
	sessionID,
	accountID,
	deviceID,
	refreshHash string,
	expiresAt time.Time,
) *aggregate.SessionAggregate {
	t.Helper()

	sessionAgg, err := aggregate.NewSessionAggregate(sessionID)
	if err != nil {
		t.Fatalf("NewSessionAggregate() error = %v", err)
	}
	if err := sessionAgg.Restore(newActiveSession(sessionID, accountID, deviceID, refreshHash, expiresAt)); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	return sessionAgg
}

func newKnownDeviceAggregate(t *testing.T, accountID, deviceID, deviceUID string) *aggregate.DeviceAggregate {
	t.Helper()

	deviceAgg, err := aggregate.NewDeviceAggregate(deviceID)
	if err != nil {
		t.Fatalf("NewDeviceAggregate() error = %v", err)
	}
	if err := deviceAgg.Restore(newKnownDevice(accountID, deviceID, deviceUID)); err != nil {
		t.Fatalf("Restore() error = %v", err)
	}

	return deviceAgg
}
