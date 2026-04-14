package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	appCtx "go-socket/core/context"
	"go-socket/core/modules/account/domain/entity"
	sharedcache "go-socket/core/shared/infra/cache"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

const emailVerificationTTL = 15 * time.Minute

var ErrVerificationTokenInvalid = errors.New("verification token is invalid or expired")

type EmailVerificationTokenPayload struct {
	AccountID string    `json:"account_id"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

//go:generate mockgen -package=service -destination=email_verification_service_mock.go -source=email_verification_service.go
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

//go:generate mockgen -package=service -destination=email_verification_service_mock.go -source=email_verification_service.go
type EmailVerificationService interface {
	SendVerificationEmail(ctx context.Context, account *entity.Account, now time.Time) (string, time.Time, error)
	ConsumeVerificationToken(ctx context.Context, token string) (*EmailVerificationTokenPayload, error)
}

type emailVerificationService struct {
	cache sharedcache.Cache
	smtp  Mailer
}

func NewEmailVerificationService(appCtx *appCtx.AppContext) EmailVerificationService {
	return &emailVerificationService{
		cache: appCtx.GetCache(),
		smtp:  appCtx.GetSMTP(),
	}
}

func (s *emailVerificationService) SendVerificationEmail(ctx context.Context, account *entity.Account, now time.Time) (string, time.Time, error) {
	if account == nil {
		return "", time.Time{}, stackErr.Error(errors.New("account is nil"))
	}

	requestedAt := normalizeVerificationTime(now)
	token := uuid.NewString()
	expiresAt := requestedAt.Add(emailVerificationTTL)

	payload := EmailVerificationTokenPayload{
		AccountID: account.ID,
		Email:     account.Email.Value(),
		ExpiresAt: expiresAt,
	}

	if err := s.cache.SetObject(ctx, emailVerificationCacheKey(token), payload, emailVerificationTTL); err != nil {
		return "", time.Time{}, stackErr.Error(err)
	}

	subject := "Verify your email"
	body := fmt.Sprintf("Use this token to verify your email: %s\nExpires at: %s", token, expiresAt.Format(time.RFC3339))
	if err := s.smtp.Send(ctx, account.Email.Value(), subject, body); err != nil {
		return "", time.Time{}, stackErr.Error(err)
	}

	return token, expiresAt, nil
}

func (s *emailVerificationService) ConsumeVerificationToken(ctx context.Context, token string) (*EmailVerificationTokenPayload, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, stackErr.Error(ErrVerificationTokenInvalid)
	}

	key := emailVerificationCacheKey(token)
	if s.cache.Exists(ctx, key) == 0 {
		return nil, stackErr.Error(ErrVerificationTokenInvalid)
	}

	data, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var payload EmailVerificationTokenPayload
	if err := json.Unmarshal(data, &payload); err != nil {
		return nil, stackErr.Error(err)
	}

	if err := s.cache.Delete(ctx, key); err != nil {
		return nil, stackErr.Error(err)
	}

	if payload.AccountID == "" || payload.Email == "" || payload.ExpiresAt.IsZero() || time.Now().UTC().After(payload.ExpiresAt.UTC()) {
		return nil, stackErr.Error(ErrVerificationTokenInvalid)
	}

	return &payload, nil
}

func emailVerificationCacheKey(token string) string {
	return "account:verify_email:" + token
}

func normalizeVerificationTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}
