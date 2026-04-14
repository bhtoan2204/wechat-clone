package repos

import (
	"context"

	"go-socket/core/modules/payment/domain/entity"
	eventpkg "go-socket/core/shared/pkg/event"
)

//go:generate mockgen -package=repos -destination=provider_payment_repo_mock.go -source=provider_payment_repo.go
type ProviderPaymentRepository interface {
	CreatePaymentIntent(ctx context.Context, intent *entity.PaymentIntent, createdEvent eventpkg.Event) error
	SavePaymentIntent(ctx context.Context, intent *entity.PaymentIntent, outboxEvents ...eventpkg.Event) error
	FinalizeSuccessfulPayment(
		ctx context.Context,
		intent *entity.PaymentIntent,
		processedEvent *entity.ProcessedPaymentEvent,
		successEvent eventpkg.Event,
		outboxEvents ...eventpkg.Event,
	) error

	GetIntentByTransactionID(ctx context.Context, transactionID string) (*entity.PaymentIntent, error)
	GetIntentByExternalRef(ctx context.Context, provider, externalRef string) (*entity.PaymentIntent, error)
	IsProcessed(ctx context.Context, provider, idempotencyKey string) (bool, error)
}
