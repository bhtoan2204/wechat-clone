package xpaseto

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"go-socket/core/modules/account/domain/entity"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/o1egl/paseto"
	"go.uber.org/zap"
)

type PasetoPayload struct {
	AccountID string
	Email     string
	ExpiresAt time.Time
}

type PasetoService interface {
	GenerateToken(ctx context.Context, account *entity.Account) (string, time.Time, error)
	ParseToken(ctx context.Context, token string) (*PasetoPayload, error)
}

type pasetoService struct {
	paseto     *paseto.V2
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	issuer     string
	ttl        time.Duration
}

func NewPaseto(cfg *config.Config) (PasetoService, error) {
	privateKeyBytes, err := base64.StdEncoding.DecodeString(cfg.AuthConfig.PrivateKey)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(cfg.AuthConfig.PublicKey)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	privateKey := ed25519.PrivateKey(privateKeyBytes)
	publicKey := ed25519.PublicKey(publicKeyBytes)

	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, stackErr.Error(fmt.Errorf("invalid private key size"))
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return nil, stackErr.Error(fmt.Errorf("invalid public key size"))
	}

	if cfg.AuthConfig.AccessTokenTTLSeconds <= 0 {
		return nil, stackErr.Error(fmt.Errorf("token ttl must be positive"))
	}

	return &pasetoService{
		paseto:     paseto.NewV2(),
		publicKey:  publicKey,
		privateKey: privateKey,
		issuer:     cfg.AuthConfig.TokenIssuer,
		ttl:        time.Duration(cfg.AuthConfig.AccessTokenTTLSeconds) * time.Second,
	}, nil
}

func (p *pasetoService) GenerateToken(ctx context.Context, account *entity.Account) (string, time.Time, error) {
	if account == nil {
		return "", time.Time{}, stackErr.Error(fmt.Errorf("account is nil"))
	}
	now := time.Now().UTC()
	exp := now.Add(p.ttl).UTC()
	payload := paseto.JSONToken{
		Issuer:     p.issuer,
		Subject:    account.ID,
		IssuedAt:   now.UTC(),
		Expiration: exp.UTC(),
	}
	payload.Set("email", account.Email.Value())

	token, err := p.paseto.Sign(p.privateKey, payload, nil)
	if err != nil {
		return "", time.Time{}, stackErr.Error(err)
	}
	return token, exp, nil
}

func (p *pasetoService) ParseToken(ctx context.Context, token string) (*PasetoPayload, error) {
	logger := logging.FromContext(ctx)

	var jsonToken paseto.JSONToken

	if err := p.paseto.Verify(token, p.publicKey, &jsonToken, nil); err != nil {
		logger.Errorw("Parse token failed", zap.Error(err))
		return nil, stackErr.Error(err)
	}

	if !jsonToken.Expiration.IsZero() && time.Now().UTC().After(jsonToken.Expiration.UTC()) {
		return nil, stackErr.Error(errors.New("token expired"))
	}

	email := jsonToken.Get("email")

	return &PasetoPayload{
		AccountID: jsonToken.Subject,
		Email:     email,
		ExpiresAt: jsonToken.Expiration,
	}, nil
}
