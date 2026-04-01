package repos

import (
	"context"

	"go-socket/core/modules/ledger/domain/entity"
)

type PaymentRepository interface {
	CreateIntent(ctx context.Context, intent *entity.PaymentIntent) error
	GetIntentByTransactionID(ctx context.Context, transactionID string) (*entity.PaymentIntent, error)
	GetIntentByExternalRef(ctx context.Context, provider, externalRef string) (*entity.PaymentIntent, error)
	UpdateIntentProviderState(ctx context.Context, transactionID, externalRef, status string) error
	UpdateIntentStatus(ctx context.Context, transactionID, status string) error
	IsProcessed(ctx context.Context, provider, idempotencyKey string) (bool, error)
	MarkProcessed(ctx context.Context, event *entity.ProcessedPaymentEvent) error
}
