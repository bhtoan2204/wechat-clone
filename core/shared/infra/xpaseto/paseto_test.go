package xpaseto

import (
	"context"
	"go-socket/core/modules/account/domain/entity"
	"go-socket/core/shared/config"
	"testing"
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
