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
)

func TestAuthenticationService_Authenticate(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	email, err := valueobject.NewEmail("alice@example.com")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}

	passwordHash, err := valueobject.NewHashedPassword("hashed-password")
	if err != nil {
		t.Fatalf("NewHashedPassword() error = %v", err)
	}

	accountAggregate, err := aggregate.NewAccountAggregate("acc-1")
	if err != nil {
		t.Fatalf("NewAccountAggregate() error = %v", err)
	}

	now := time.Date(2026, time.April, 14, 8, 0, 0, 0, time.UTC)
	if err := accountAggregate.Register(email, passwordHash, "Alice", now); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	expiresAt := time.Date(2026, time.April, 14, 11, 0, 0, 0, time.UTC)

	baseRepo := repos.NewMockRepos(ctrl)
	hasher := hasher.NewMockHasher(ctrl)
	paseto := xpaseto.NewMockPasetoService(ctrl)

	hasher.
		EXPECT().
		Verify(gomock.Any(), "password123", "hashed-password").
		Return(true, nil)

	paseto.
		EXPECT().
		GenerateAccessToken(gomock.Any(), gomock.AssignableToTypeOf(&entity.Account{})).
		DoAndReturn(func(_ context.Context, account *entity.Account) (string, time.Time, error) {
			if account == nil || account.ID != "acc-1" {
				t.Fatalf("expected account snapshot from aggregate")
			}
			return "signed-token", expiresAt, nil
		})

	service := &authenticationService{
		baseRepo: baseRepo,
		hasher:   hasher,
		paseto:   paseto,
	}

	result, err := service.Authenticate(context.Background(), AuthenticateAccountCommand{
		Email:    "alice@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}

	if result.AccessToken != "signed-token" {
		t.Fatalf("expected signed-token, got %q", result.AccessToken)
	}

	if !result.AccessExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expiresAt %v, got %v", expiresAt, result.AccessExpiresAt)
	}
}

func TestAuthenticationService_Authenticate_MapsNotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	baseRepo := repos.NewMockRepos(ctrl)
	hasher := hasher.NewMockHasher(ctrl)
	paseto := xpaseto.NewMockPasetoService(ctrl)

	service := &authenticationService{
		baseRepo: baseRepo,
		hasher:   hasher,
		paseto:   paseto,
	}

	_, err := service.Authenticate(context.Background(), AuthenticateAccountCommand{
		Email:    "missing@example.com",
		Password: "password123",
	})

	if !errors.Is(err, ErrAuthenticationAccountNotFound) {
		t.Fatalf("expected ErrAuthenticationAccountNotFound, got %v", err)
	}
}

func TestAuthenticationService_Authenticate_MapsInvalidPassword(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	email, err := valueobject.NewEmail("alice@example.com")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}

	passwordHash, err := valueobject.NewHashedPassword("hashed-password")
	if err != nil {
		t.Fatalf("NewHashedPassword() error = %v", err)
	}

	accountAggregate, err := aggregate.NewAccountAggregate("acc-1")
	if err != nil {
		t.Fatalf("NewAccountAggregate() error = %v", err)
	}

	now := time.Date(2026, time.April, 14, 8, 0, 0, 0, time.UTC)
	if err := accountAggregate.Register(email, passwordHash, "Alice", now); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	baseRepo := repos.NewMockRepos(ctrl)
	hasher := hasher.NewMockHasher(ctrl)
	paseto := xpaseto.NewMockPasetoService(ctrl)

	hasher.
		EXPECT().
		Verify(gomock.Any(), "password123", "hashed-password").
		Return(false, nil)

	service := &authenticationService{
		baseRepo: baseRepo,
		hasher:   hasher,
		paseto:   paseto,
	}

	_, err = service.Authenticate(context.Background(), AuthenticateAccountCommand{
		Email:    "alice@example.com",
		Password: "password123",
	})

	if !errors.Is(err, ErrAuthenticationInvalidPassword) {
		t.Fatalf("expected ErrAuthenticationInvalidPassword, got %v", err)
	}
}
