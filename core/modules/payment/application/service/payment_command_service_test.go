package service

import (
	"context"
	"errors"
	"testing"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/shared/pkg/actorctx"
)

func TestCreatePaymentRejectsClientManagedAccounts(t *testing.T) {
	t.Run("rejects system managed debit account override", func(t *testing.T) {
		svc := &paymentCommandService{}

		_, err := svc.CreatePayment(
			actorctx.WithActor(context.Background(), actorctx.Actor{AccountID: "acc-user-1"}),
			&in.CreatePaymentRequest{
				Provider: "stripe",
				Amount:   100,
				Currency: "VND",
				// DebitAccountID: "wallet:anything",
			},
		)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("rejects credit account that does not match authenticated actor", func(t *testing.T) {
		svc := &paymentCommandService{}

		_, err := svc.CreatePayment(
			actorctx.WithActor(context.Background(), actorctx.Actor{AccountID: "acc-user-1"}),
			&in.CreatePaymentRequest{
				Provider:        "stripe",
				Amount:          100,
				Currency:        "VND",
				CreditAccountID: "acc-user-2",
			},
		)
		if !errors.Is(err, ErrValidation) {
			t.Fatalf("expected validation error, got %v", err)
		}
	})

	t.Run("requires authenticated actor for create payment", func(t *testing.T) {
		svc := &paymentCommandService{}

		_, err := svc.CreatePayment(context.Background(), &in.CreatePaymentRequest{
			Provider: "stripe",
			Amount:   100,
			Currency: "VND",
		})
		if !errors.Is(err, ErrPaymentUnauthorized) {
			t.Fatalf("expected unauthorized error, got %v", err)
		}
	})
}
