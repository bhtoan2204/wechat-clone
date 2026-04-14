package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-socket/core/modules/account/domain/repos"
	"go-socket/core/shared/infra/xpaseto"
	"go-socket/core/shared/pkg/hasher"

	gomock "go.uber.org/mock/gomock"
)

func TestRegistrationServiceRegister(t *testing.T) {
	t.Parallel()
	mockCtrl := gomock.NewController(t)
	mockBaseRepo := repos.NewMockRepos(mockCtrl)
	expiresAt := time.Date(2026, time.April, 14, 10, 0, 0, 0, time.UTC)
	service := &registrationService{
		baseRepo: mockBaseRepo,
		hasher:   &hasher.MockHasher{},
		paseto:   &xpaseto.MockPasetoService{},
	}

	result, err := service.Register(context.Background(), RegisterAccountCommand{
		Email:       "alice@example.com",
		Password:    "password123",
		DisplayName: "Alice",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.AccessToken != "signed-token" {
		t.Fatalf("expected signed token, got %q", result.AccessToken)
	}
	if !result.AccessExpiresAt.Equal(expiresAt) {
		t.Fatalf("expected expiresAt %v, got %v", expiresAt, result.AccessExpiresAt)
	}
}

func TestRegistrationServiceRegisterReturnsAccountExists(t *testing.T) {
	t.Parallel()

	service := &registrationService{
		baseRepo: &repos.MockRepos{},
		hasher:   &hasher.MockHasher{},
		paseto:   &xpaseto.MockPasetoService{},
	}

	_, err := service.Register(context.Background(), RegisterAccountCommand{
		Email:       "alice@example.com",
		Password:    "password123",
		DisplayName: "Alice",
	})
	if !errors.Is(err, ErrRegistrationAccountExists) {
		t.Fatalf("expected ErrRegistrationAccountExists, got %v", err)
	}
}
