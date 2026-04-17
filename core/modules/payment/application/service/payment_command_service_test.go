package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go-socket/core/modules/payment/application/dto/in"
	paymentaggregate "go-socket/core/modules/payment/domain/aggregate"
	"go-socket/core/modules/payment/domain/entity"
	repos "go-socket/core/modules/payment/domain/repos"
	domainservice "go-socket/core/modules/payment/domain/service"
	sharedlock "go-socket/core/shared/infra/lock"
	"go-socket/core/shared/pkg/actorctx"

	"go.uber.org/mock/gomock"
)

func TestCreatePaymentRejectsUnauthorizedOrCrossAccountRequests(t *testing.T) {
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

func TestProcessWebhookLocksByTransactionIDAndFinalizesSuccess(t *testing.T) {
	ctrl := gomock.NewController(t)
	baseRepo := repos.NewMockRepos(ctrl)
	txRepos := repos.NewMockRepos(ctrl)
	providerRepo := repos.NewMockProviderPaymentRepository(ctrl)
	providerRegistry := domainservice.NewMockPaymentProviderRegistry(ctrl)
	provider := domainservice.NewMockPaymentProvider(ctrl)
	locker := sharedlock.NewMockLock(ctrl)

	paymentAggregate := mustRehydratePaymentAggregate(t, "txn-1", "stripe", 100, "VND", "wallet:available")

	providerRegistry.EXPECT().Get("stripe").Return(provider, nil)
	provider.EXPECT().ParseWebhook(gomock.Any(), []byte("{}"), "sig-1").Return(&domainservice.PaymentWebhook{
		Provider: "stripe",
		Result: entity.PaymentProviderResult{
			TransactionID: "txn-1",
			EventID:       "evt-1",
			EventType:     "checkout.session.completed",
			Status:        entity.PaymentStatusSuccess,
			Amount:        100,
			Currency:      "VND",
			ExternalRef:   "cs-1",
		},
	}, nil)

	baseRepo.EXPECT().ProviderPaymentRepository().Return(providerRepo).AnyTimes()
	providerRepo.EXPECT().GetByTransactionID(gomock.Any(), "txn-1").Return(paymentAggregate, nil).Times(2)
	locker.EXPECT().AcquireLock(gomock.Any(), "payment:txn-1", gomock.Any(), 30*time.Second, 100*time.Millisecond, 3*time.Second).Return(true, nil)
	locker.EXPECT().ReleaseLock(gomock.Any(), "payment:txn-1", gomock.Any()).Return(true, nil)
	baseRepo.EXPECT().WithTransaction(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(repos.Repos) error) error {
		return fn(txRepos)
	})
	txRepos.EXPECT().ProviderPaymentRepository().Return(providerRepo).AnyTimes()
	providerRepo.EXPECT().Save(gomock.Any(), paymentAggregate).DoAndReturn(func(_ context.Context, savedAggregate *paymentaggregate.PaymentIntentAggregate) error {
		if savedAggregate.Status() != entity.PaymentStatusSuccess {
			t.Fatalf("expected saved aggregate status success, got %s", savedAggregate.Status())
		}
		processedEvents := savedAggregate.PendingProcessedEvents()
		if len(processedEvents) != 1 {
			t.Fatalf("expected 1 processed event, got %d", len(processedEvents))
		}
		if processedEvents[0].IdempotencyKey != "payment.succeeded:txn-1" {
			t.Fatalf("unexpected processed event idempotency key: %s", processedEvents[0].IdempotencyKey)
		}
		outboxEvents := savedAggregate.PendingOutboxEvents()
		if len(outboxEvents) != 1 {
			t.Fatalf("expected 1 outbox event, got %d", len(outboxEvents))
		}
		if outboxEvents[0].EventName != "payment.succeeded" {
			t.Fatalf("unexpected outbox event name: %s", outboxEvents[0].EventName)
		}
		return nil
	})

	svc := &paymentCommandService{
		baseRepo:         baseRepo,
		locker:           locker,
		providerRegistry: providerRegistry,
	}

	response, err := svc.ProcessWebhook(context.Background(), &in.ProcessWebhookRequest{
		Provider:  "stripe",
		Signature: "sig-1",
		Payload:   "{}",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if response.TransactionID != "txn-1" {
		t.Fatalf("unexpected transaction id: %s", response.TransactionID)
	}
	if response.Status != entity.PaymentStatusSuccess {
		t.Fatalf("unexpected status: %s", response.Status)
	}
	if response.Duplicate {
		t.Fatalf("expected non-duplicate webhook processing")
	}
}

func TestApplyProviderOutcomeFinalizesSuccessOnlyOncePerPayment(t *testing.T) {
	ctrl := gomock.NewController(t)
	baseRepo := repos.NewMockRepos(ctrl)
	txRepos := repos.NewMockRepos(ctrl)
	providerRepo := repos.NewMockProviderPaymentRepository(ctrl)

	paymentAggregate := mustRehydratePaymentAggregate(t, "txn-1", "stripe", 100, "VND", "wallet:available")

	baseRepo.EXPECT().WithTransaction(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(repos.Repos) error) error {
		return fn(txRepos)
	}).Times(1)
	txRepos.EXPECT().ProviderPaymentRepository().Return(providerRepo).AnyTimes()
	providerRepo.EXPECT().Save(gomock.Any(), paymentAggregate).DoAndReturn(func(_ context.Context, savedAggregate *paymentaggregate.PaymentIntentAggregate) error {
		if savedAggregate.Status() != entity.PaymentStatusSuccess {
			t.Fatalf("expected saved aggregate status success, got %s", savedAggregate.Status())
		}
		processedEvents := savedAggregate.PendingProcessedEvents()
		if len(processedEvents) != 1 {
			t.Fatalf("expected 1 processed event, got %d", len(processedEvents))
		}
		if processedEvents[0].IdempotencyKey != "payment.succeeded:txn-1" {
			t.Fatalf("unexpected processed event idempotency key: %s", processedEvents[0].IdempotencyKey)
		}
		return nil
	}).Times(1)

	svc := &paymentCommandService{baseRepo: baseRepo}
	duplicate, err := svc.applyProviderOutcome(context.Background(), paymentAggregate, entity.PaymentProviderResult{
		TransactionID: "txn-1",
		EventID:       "evt-checkout-completed",
		EventType:     "checkout.session.completed",
		Status:        entity.PaymentStatusSuccess,
		Amount:        100,
		Currency:      "VND",
		ExternalRef:   "cs-1",
	}, "", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if duplicate {
		t.Fatalf("expected first success to finalize payment")
	}
	if paymentAggregate.Status() != entity.PaymentStatusSuccess {
		t.Fatalf("expected success status, got %s", paymentAggregate.Status())
	}

	duplicate, err = svc.applyProviderOutcome(context.Background(), paymentAggregate, entity.PaymentProviderResult{
		TransactionID: "txn-1",
		EventID:       "evt-charge-succeeded",
		EventType:     "charge.succeeded",
		Status:        entity.PaymentStatusSuccess,
		Amount:        100,
		Currency:      "VND",
		ExternalRef:   "cs-1",
	}, "", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !duplicate {
		t.Fatalf("expected second success to be treated as duplicate")
	}
	if paymentAggregate.Status() != entity.PaymentStatusSuccess {
		t.Fatalf("expected status to stay success, got %s", paymentAggregate.Status())
	}
}

func TestApplyProviderOutcomeIgnoresFailAfterSuccessWithoutPersist(t *testing.T) {
	paymentAggregate := mustRehydratePaymentAggregate(t, "txn-1", "stripe", 100, "VND", "wallet:available")
	_, err := paymentAggregate.ApplyProviderOutcome(entity.PaymentProviderResult{
		ExternalRef: "cs-1",
		Status:      entity.PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
	}, "", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("apply initial success: %v", err)
	}
	paymentAggregate.MarkPersisted()

	svc := &paymentCommandService{}
	duplicate, err := svc.applyProviderOutcome(context.Background(), paymentAggregate, entity.PaymentProviderResult{
		TransactionID: "txn-1",
		EventID:       "evt-payment-failed",
		EventType:     "payment_intent.payment_failed",
		Status:        entity.PaymentStatusFailed,
		Amount:        100,
		Currency:      "VND",
		ExternalRef:   "cs-1",
	}, "", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !duplicate {
		t.Fatalf("expected fail-after-success to be ignored as duplicate")
	}
	if paymentAggregate.Status() != entity.PaymentStatusSuccess {
		t.Fatalf("expected status to stay success, got %s", paymentAggregate.Status())
	}
}

func TestApplyProviderOutcomeFinalizesRefundAsReversal(t *testing.T) {
	ctrl := gomock.NewController(t)
	baseRepo := repos.NewMockRepos(ctrl)
	txRepos := repos.NewMockRepos(ctrl)
	providerRepo := repos.NewMockProviderPaymentRepository(ctrl)

	paymentAggregate := mustRehydratePaymentAggregate(t, "txn-1", "stripe", 100, "VND", "wallet:available")
	_, err := paymentAggregate.ApplyProviderOutcome(entity.PaymentProviderResult{
		ExternalRef: "cs-1",
		Status:      entity.PaymentStatusSuccess,
		Amount:      100,
		Currency:    "VND",
	}, "", false, time.Now().UTC())
	if err != nil {
		t.Fatalf("apply initial success: %v", err)
	}
	paymentAggregate.MarkPersisted()

	baseRepo.EXPECT().WithTransaction(gomock.Any(), gomock.Any()).DoAndReturn(func(ctx context.Context, fn func(repos.Repos) error) error {
		return fn(txRepos)
	})
	txRepos.EXPECT().ProviderPaymentRepository().Return(providerRepo).AnyTimes()
	providerRepo.EXPECT().Save(gomock.Any(), paymentAggregate).DoAndReturn(func(_ context.Context, savedAggregate *paymentaggregate.PaymentIntentAggregate) error {
		if savedAggregate.Status() != entity.PaymentStatusRefunded {
			t.Fatalf("expected refunded status, got %s", savedAggregate.Status())
		}
		processedEvents := savedAggregate.PendingProcessedEvents()
		if len(processedEvents) != 1 {
			t.Fatalf("expected 1 processed event, got %d", len(processedEvents))
		}
		if processedEvents[0].IdempotencyKey != "payment.refunded:txn-1" {
			t.Fatalf("unexpected processed event idempotency key: %s", processedEvents[0].IdempotencyKey)
		}
		outboxEvents := savedAggregate.PendingOutboxEvents()
		if len(outboxEvents) != 1 {
			t.Fatalf("expected 1 outbox event, got %d", len(outboxEvents))
		}
		if outboxEvents[0].EventName != "payment.refunded" {
			t.Fatalf("unexpected outbox event name: %s", outboxEvents[0].EventName)
		}
		return nil
	})

	svc := &paymentCommandService{baseRepo: baseRepo}
	duplicate, err := svc.applyProviderOutcome(context.Background(), paymentAggregate, entity.PaymentProviderResult{
		TransactionID: "txn-1",
		EventID:       "evt-charge-refunded",
		EventType:     "charge.refunded",
		Status:        entity.PaymentStatusRefunded,
		Amount:        100,
		Currency:      "VND",
		ExternalRef:   "cs-1",
	}, "", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if duplicate {
		t.Fatalf("expected first refund to finalize reversal")
	}
	if paymentAggregate.Status() != entity.PaymentStatusRefunded {
		t.Fatalf("expected refunded status, got %s", paymentAggregate.Status())
	}
}

func mustRehydratePaymentAggregate(
	t *testing.T,
	transactionID string,
	provider string,
	amount int64,
	currency string,
	creditAccountID string,
) *paymentaggregate.PaymentIntentAggregate {
	t.Helper()

	intent, err := entity.NewProviderTopUpIntent(transactionID, provider, amount, currency, creditAccountID, time.Now().UTC())
	if err != nil {
		t.Fatalf("new provider top up intent: %v", err)
	}
	paymentAggregate, err := paymentaggregate.RestorePaymentIntentAggregate(intent)
	if err != nil {
		t.Fatalf("rehydrate payment aggregate: %v", err)
	}
	return paymentAggregate
}
