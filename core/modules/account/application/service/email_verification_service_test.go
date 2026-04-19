package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"wechat-clone/core/modules/account/domain/entity"
	valueobject "wechat-clone/core/modules/account/domain/value_object"
	sharedcache "wechat-clone/core/shared/infra/cache"

	"go.uber.org/mock/gomock"
)

func TestEmailVerificationService_SendVerificationEmail_UsesTemplateAndStoresToken(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := sharedcache.NewMockCache(ctrl)
	mailer := NewMockMailer(ctrl)

	email, err := valueobject.NewEmail("alice@example.com")
	if err != nil {
		t.Fatalf("NewEmail() error = %v", err)
	}

	account := &entity.Account{
		ID:          "acc-1",
		Email:       email,
		DisplayName: "Alice",
	}

	cache.EXPECT().
		SetObject(gomock.Any(), gomock.Any(), gomock.Any(), emailVerificationTTL).
		DoAndReturn(func(_ context.Context, key string, val interface{}, duration time.Duration) error {
			if key == "" || len(key) <= len("account:verify_email:") || key[:len("account:verify_email:")] != "account:verify_email:" {
				t.Fatalf("expected verification cache key, got %q", key)
			}
			payload, ok := val.(EmailVerificationTokenPayload)
			if !ok {
				t.Fatalf("expected EmailVerificationTokenPayload, got %T", val)
			}
			if payload.AccountID != "acc-1" || payload.Email != "alice@example.com" {
				t.Fatalf("unexpected payload %+v", payload)
			}
			if duration != emailVerificationTTL {
				t.Fatalf("expected ttl %v, got %v", emailVerificationTTL, duration)
			}
			return nil
		})

	mailer.EXPECT().
		SendTemplate(gomock.Any(), "alice@example.com", "Verify your email", "verify_email.html", gomock.Any()).
		DoAndReturn(func(_ context.Context, to, subject, templateName string, data any) error {
			templateData, ok := data.(verifyEmailTemplateData)
			if !ok {
				t.Fatalf("expected verifyEmailTemplateData, got %T", data)
			}
			if templateData.DisplayName != "Alice" {
				t.Fatalf("expected display name Alice, got %q", templateData.DisplayName)
			}
			if templateData.Email != "alice@example.com" {
				t.Fatalf("expected email alice@example.com, got %q", templateData.Email)
			}
			if templateData.Token == "" {
				t.Fatalf("expected non-empty token")
			}
			if templateData.VerificationURL == "" {
				t.Fatalf("expected verification url to be built")
			}
			return nil
		})

	service := &emailVerificationService{
		cache:          cache,
		smtp:           mailer,
		verifyEmailURL: "http://localhost:5173/verify-email",
	}

	token, expiresAt, err := service.SendVerificationEmail(context.Background(), account)
	if err != nil {
		t.Fatalf("SendVerificationEmail() error = %v", err)
	}
	if token == "" {
		t.Fatalf("expected non-empty token")
	}
	if expiresAt.IsZero() {
		t.Fatalf("expected non-zero expiry")
	}
}

func TestEmailVerificationService_ConsumeVerificationToken_DeletesKeyAfterSuccess(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	cache := sharedcache.NewMockCache(ctrl)
	expiresAt := time.Now().UTC().Add(10 * time.Minute)
	payload := EmailVerificationTokenPayload{
		AccountID: "acc-1",
		Email:     "alice@example.com",
		ExpiresAt: expiresAt,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	cache.EXPECT().Exists(gomock.Any(), "account:verify_email:token-1").Return(int64(1))
	cache.EXPECT().Get(gomock.Any(), "account:verify_email:token-1").Return(raw, nil)
	cache.EXPECT().Delete(gomock.Any(), "account:verify_email:token-1").Return(nil)

	service := &emailVerificationService{cache: cache}

	result, err := service.ConsumeVerificationToken(context.Background(), "token-1")
	if err != nil {
		t.Fatalf("ConsumeVerificationToken() error = %v", err)
	}
	if result.AccountID != "acc-1" {
		t.Fatalf("expected account acc-1, got %q", result.AccountID)
	}
	if result.Email != "alice@example.com" {
		t.Fatalf("expected alice@example.com, got %q", result.Email)
	}
}
