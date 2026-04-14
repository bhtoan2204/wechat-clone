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

type TokenType string

const (
	TokenTypeAccess  TokenType = "access"
	TokenTypeRefresh TokenType = "refresh"
)

type PasetoPayload struct {
	AccountID string
	Email     string
	TokenType TokenType
	ExpiresAt time.Time
	IssuedAt  time.Time
}

//go:generate mockgen -package=xpaseto -destination=paseto_mock.go -source=paseto.go
type PasetoService interface {
	GenerateAccessToken(ctx context.Context, account *entity.Account) (string, time.Time, error)
	GenerateRefreshToken(ctx context.Context, account *entity.Account) (string, time.Time, error)

	ParseAccessToken(ctx context.Context, token string) (*PasetoPayload, error)
	ParseRefreshToken(ctx context.Context, token string) (*PasetoPayload, error)
}

type pasetoService struct {
	paseto *paseto.V2
	issuer string

	accessPublicKey  ed25519.PublicKey
	accessPrivateKey ed25519.PrivateKey
	accessTTL        time.Duration

	refreshPublicKey  ed25519.PublicKey
	refreshPrivateKey ed25519.PrivateKey
	refreshTTL        time.Duration
}

func NewPaseto(cfg *config.Config) (PasetoService, error) {
	accessPrivateKeyBytes, err := base64.StdEncoding.DecodeString(cfg.AuthConfig.AccessPrivateKey)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accessPublicKeyBytes, err := base64.StdEncoding.DecodeString(cfg.AuthConfig.AccessPublicKey)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	refreshPrivateKeyBytes, err := base64.StdEncoding.DecodeString(cfg.AuthConfig.RefreshPrivateKey)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	refreshPublicKeyBytes, err := base64.StdEncoding.DecodeString(cfg.AuthConfig.RefreshPublicKey)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	accessPrivateKey := ed25519.PrivateKey(accessPrivateKeyBytes)
	accessPublicKey := ed25519.PublicKey(accessPublicKeyBytes)
	refreshPrivateKey := ed25519.PrivateKey(refreshPrivateKeyBytes)
	refreshPublicKey := ed25519.PublicKey(refreshPublicKeyBytes)

	if len(accessPrivateKey) != ed25519.PrivateKeySize || len(accessPublicKey) != ed25519.PublicKeySize ||
		len(refreshPrivateKey) != ed25519.PrivateKeySize || len(refreshPublicKey) != ed25519.PublicKeySize {
		return nil, stackErr.Error(fmt.Errorf("invalid private key size"))
	}

	if cfg.AuthConfig.AccessTokenTTLSeconds <= 0 {
		return nil, stackErr.Error(fmt.Errorf("access token ttl must be positive"))
	}

	if cfg.AuthConfig.RefreshTokenTTLSeconds <= 0 {
		return nil, stackErr.Error(fmt.Errorf("refresh token ttl must be positive"))
	}

	return &pasetoService{
		paseto: paseto.NewV2(),
		issuer: cfg.AuthConfig.TokenIssuer,

		accessPublicKey:  accessPublicKey,
		accessPrivateKey: accessPrivateKey,
		accessTTL:        time.Duration(cfg.AuthConfig.AccessTokenTTLSeconds) * time.Second,

		refreshPublicKey:  refreshPublicKey,
		refreshPrivateKey: refreshPrivateKey,
		refreshTTL:        time.Duration(cfg.AuthConfig.RefreshTokenTTLSeconds) * time.Second,
	}, nil
}

func (p *pasetoService) GenerateAccessToken(ctx context.Context, account *entity.Account) (string, time.Time, error) {
	return p.generateToken(account, TokenTypeAccess)
}

func (p *pasetoService) GenerateRefreshToken(ctx context.Context, account *entity.Account) (string, time.Time, error) {
	return p.generateToken(account, TokenTypeRefresh)
}

func (p *pasetoService) generateToken(account *entity.Account, tokenType TokenType) (string, time.Time, error) {
	if account == nil {
		return "", time.Time{}, stackErr.Error(fmt.Errorf("account is nil"))
	}

	now := time.Now().UTC()

	var (
		exp        time.Time
		privateKey ed25519.PrivateKey
	)

	switch tokenType {
	case TokenTypeAccess:
		exp = now.Add(p.accessTTL).UTC()
		privateKey = p.accessPrivateKey
	case TokenTypeRefresh:
		exp = now.Add(p.refreshTTL).UTC()
		privateKey = p.refreshPrivateKey
	default:
		return "", time.Time{}, stackErr.Error(fmt.Errorf("invalid token type"))
	}

	payload := paseto.JSONToken{
		Issuer:     p.issuer,
		Subject:    account.ID,
		IssuedAt:   now,
		Expiration: exp,
	}

	payload.Set("email", account.Email.Value())
	payload.Set("token_use", string(tokenType))

	token, err := p.paseto.Sign(privateKey, payload, nil)
	if err != nil {
		return "", time.Time{}, stackErr.Error(err)
	}

	return token, exp, nil
}

func (p *pasetoService) ParseAccessToken(ctx context.Context, token string) (*PasetoPayload, error) {
	return p.parseToken(ctx, token, TokenTypeAccess)
}

func (p *pasetoService) ParseRefreshToken(ctx context.Context, token string) (*PasetoPayload, error) {
	return p.parseToken(ctx, token, TokenTypeRefresh)
}

func (p *pasetoService) parseToken(ctx context.Context, token string, expectedType TokenType) (*PasetoPayload, error) {
	logger := logging.FromContext(ctx)

	var (
		jsonToken paseto.JSONToken
		publicKey ed25519.PublicKey
	)

	switch expectedType {
	case TokenTypeAccess:
		publicKey = p.accessPublicKey
	case TokenTypeRefresh:
		publicKey = p.refreshPublicKey
	default:
		return nil, stackErr.Error(fmt.Errorf("invalid expected token type"))
	}

	if err := p.paseto.Verify(token, publicKey, &jsonToken, nil); err != nil {
		logger.Errorw("Verify token failed", zap.String("expected_type", string(expectedType)), zap.Error(err))
		return nil, stackErr.Error(err)
	}

	if jsonToken.Issuer != p.issuer {
		return nil, stackErr.Error(errors.New("invalid token issuer"))
	}

	if jsonToken.Subject == "" {
		return nil, stackErr.Error(errors.New("invalid token subject"))
	}

	if !jsonToken.Expiration.IsZero() && time.Now().UTC().After(jsonToken.Expiration.UTC()) {
		return nil, stackErr.Error(errors.New("token expired"))
	}

	tokenUse := jsonToken.Get("token_use")
	if tokenUse == "" {
		return nil, stackErr.Error(errors.New("missing token_use"))
	}

	if TokenType(tokenUse) != expectedType {
		return nil, stackErr.Error(errors.New("invalid token type"))
	}

	email := jsonToken.Get("email")

	return &PasetoPayload{
		AccountID: jsonToken.Subject,
		Email:     email,
		TokenType: expectedType,
		ExpiresAt: jsonToken.Expiration,
		IssuedAt:  jsonToken.IssuedAt,
	}, nil
}
