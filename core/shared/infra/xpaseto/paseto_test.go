package xpaseto

import (
	"context"
	"testing"
	"time"
	"wechat-clone/core/modules/account/domain/entity"
	valueobject "wechat-clone/core/modules/account/domain/value_object"
	"wechat-clone/core/shared/config"
)

func TestPaseto(t *testing.T) {
	cfg := &config.Config{
		AuthConfig: config.AuthConfig{
			TokenIssuer:            "chat",
			AccessTokenTTLSeconds:  9000,
			RefreshTokenTTLSeconds: 26400,
			AccessPublicKey:        "g2NuXbGMgDnw04S8KmeKqJJ94WwABPoe/2HB66V1+QM=",
			AccessPrivateKey:       "OghFb8xO1EqyzKRc1/q7hgAkNzZfZJXOkczIoey2+ViDY25dsYyAOfDThLwqZ4qokn3hbAAE+h7/YcHrpXX5Aw==",
			RefreshPublicKey:       "g2NuXbGMgDnw04S8KmeKqJJ94WwABPoe/2HB66V1+QM=",
			RefreshPrivateKey:      "OghFb8xO1EqyzKRc1/q7hgAkNzZfZJXOkczIoey2+ViDY25dsYyAOfDThLwqZ4qokn3hbAAE+h7/YcHrpXX5Aw==",
		},
	}
	pasetoSvc, err := NewPaseto(cfg)
	if err != nil {
		t.Fatal(err)
	}
	str, _, _ := pasetoSvc.GenerateAccessToken(context.Background(), &entity.Account{
		ID: "test-abc-001",
	})
	claims, err := pasetoSvc.ParseAccessToken(context.Background(), str)
	if err != nil {
		t.Fatal(err)
	}
	t.Log(claims)
}

func TestParseAccessToken(t *testing.T) {
	cfg := &config.Config{
		AuthConfig: config.AuthConfig{
			TokenIssuer:            "chat",
			AccessTokenTTLSeconds:  9000,
			RefreshTokenTTLSeconds: 26400,
			AccessPublicKey:        "vSKvNvjpCS3teuTBeXm9gHYSIGLaovZoM+vMnyNeFKk=",
			AccessPrivateKey:       "CncqpMFMEHuK1As2dIRECZ2qLZJAqgJKZmP9KdN+vLO9Iq82+OkJLe165MF5eb2AdhIgYtqi9mgz68yfI14UqQ==",
			RefreshPublicKey:       "g2NuXbGMgDnw04S8KmeKqJJ94WwABPoe/2HB66V1+QM=",
			RefreshPrivateKey:      "OghFb8xO1EqyzKRc1/q7hgAkNzZfZJXOkczIoey2+ViDY25dsYyAOfDThLwqZ4qokn3hbAAE+h7/YcHrpXX5Aw==",
		},
	}

	pasetoSvc, err := NewPaseto(cfg)
	if err != nil {
		t.Fatal(err)
	}
	email, err := valueobject.NewEmail("banhhaotoan2002@gmail.com")
	if err != nil {
		t.Fatal(err)
	}
	account := &entity.Account{
		ID:    "81636f7a-e0af-4ab2-a749-604bdd5a5cd4",
		Email: email,
	}

	token, exp, err := pasetoSvc.GenerateAccessToken(context.Background(), account)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(token)

	claims, err := pasetoSvc.ParseAccessToken(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}

	if claims.AccountID != account.ID {
		t.Fatalf("expected account id %q, got %q", account.ID, claims.AccountID)
	}

	if claims.Email != account.Email.Value() {
		t.Fatalf("expected email %q, got %q", account.Email.Value(), claims.Email)
	}

	if claims.TokenType != TokenTypeAccess {
		t.Fatalf("expected token type %q, got %q", TokenTypeAccess, claims.TokenType)
	}

	if claims.SessionID != "" {
		t.Fatalf("expected empty session id for access token, got %q", claims.SessionID)
	}

	if claims.DeviceID != "" {
		t.Fatalf("expected empty device id for access token, got %q", claims.DeviceID)
	}

	if diff := claims.ExpiresAt.Sub(exp); diff < -time.Second || diff > time.Second {
		t.Fatalf("expected exp close to %v, got %v", exp, claims.ExpiresAt)
	}

	if claims.IssuedAt.IsZero() {
		t.Fatal("expected issued at to be set")
	}
}
