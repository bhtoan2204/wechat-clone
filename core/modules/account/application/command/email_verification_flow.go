package command

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/account/domain/entity"
	sharedcache "wechat-clone/core/shared/infra/cache"
	"wechat-clone/core/shared/pkg/stackErr"
	"wechat-clone/core/shared/utils"

	"github.com/google/uuid"
)

const emailVerificationTTL = 15 * time.Minute

var ErrVerificationTokenInvalid = errors.New("verification token is invalid or expired")

type EmailVerificationTokenPayload struct {
	AccountID string    `json:"account_id"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
	SendTemplate(ctx context.Context, to, subject, templateName string, data any) error
}

type emailVerificationDependencies struct {
	cache          sharedcache.Cache
	smtp           Mailer
	verifyEmailURL string
}

func newEmailVerificationDependencies(appCtx *appCtx.AppContext) emailVerificationDependencies {
	return emailVerificationDependencies{
		cache:          appCtx.GetCache(),
		smtp:           appCtx.GetSMTP(),
		verifyEmailURL: strings.TrimSpace(appCtx.GetConfig().AuthConfig.VerifyEmailURL),
	}
}

func (s emailVerificationDependencies) SendVerificationEmail(ctx context.Context, account *entity.Account) (string, time.Time, error) {
	if account == nil {
		return "", time.Time{}, stackErr.Error(errors.New("account is nil"))
	}

	requestedAt := utils.NowUTC()
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
	templateData := verifyEmailTemplateData{
		DisplayName:      account.DisplayName,
		Email:            account.Email.Value(),
		Token:            token,
		ExpiresAt:        expiresAt.Format(time.RFC3339),
		VerificationURL:  buildVerificationURL(s.verifyEmailURL, token),
		ExpiresInMinutes: int(emailVerificationTTL / time.Minute),
	}
	if err := s.smtp.SendTemplate(ctx, account.Email.Value(), subject, "verify_email.html", templateData); err != nil {
		return "", time.Time{}, stackErr.Error(err)
	}

	return token, expiresAt, nil
}

func (s emailVerificationDependencies) ConsumeVerificationToken(ctx context.Context, token string) (*EmailVerificationTokenPayload, error) {
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

type verifyEmailTemplateData struct {
	DisplayName      string
	Email            string
	Token            string
	ExpiresAt        string
	VerificationURL  string
	ExpiresInMinutes int
}

func buildVerificationURL(baseURL, token string) string {
	baseURL = strings.TrimSpace(baseURL)
	token = strings.TrimSpace(token)
	if baseURL == "" || token == "" {
		return ""
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return ""
	}

	query := parsed.Query()
	query.Set("token", token)
	parsed.RawQuery = query.Encode()
	return parsed.String()
}
