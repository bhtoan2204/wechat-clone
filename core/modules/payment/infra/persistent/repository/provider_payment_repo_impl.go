package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	paymentaggregate "go-socket/core/modules/payment/domain/aggregate"
	"go-socket/core/modules/payment/domain/entity"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/infra/persistent/model"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

const pendingExternalRefPrefix = "__pending__:"

type providerPaymentRepoImpl struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func newProviderPaymentRepoImpl(db *gorm.DB) paymentrepos.ProviderPaymentRepository {
	return &providerPaymentRepoImpl{
		db:         db,
		serializer: eventpkg.NewSerializer(),
	}
}

func (r *providerPaymentRepoImpl) Create(ctx context.Context, aggregate *paymentaggregate.PaymentIntentAggregate) error {
	intent := normalizeProviderPaymentIntent(aggregateSnapshot(aggregate))
	if intent == nil {
		return paymentrepos.ErrProviderPaymentNotFound
	}

	if err := r.createIntent(ctx, intent); err != nil {
		return stackErr.Error(err)
	}
	if err := r.appendOutboxEvents(ctx, aggregate.PendingOutboxEvents()...); err != nil {
		return stackErr.Error(err)
	}
	aggregate.MarkPersisted()
	return nil
}

func (r *providerPaymentRepoImpl) Save(ctx context.Context, aggregate *paymentaggregate.PaymentIntentAggregate) error {
	intent := normalizeProviderPaymentIntent(aggregateSnapshot(aggregate))
	if intent == nil {
		return paymentrepos.ErrProviderPaymentNotFound
	}

	for _, processedEvent := range aggregate.PendingProcessedEvents() {
		if err := r.markProcessed(ctx, processedEvent); err != nil {
			return stackErr.Error(err)
		}
	}
	if err := r.updateIntent(ctx, intent); err != nil {
		return stackErr.Error(err)
	}
	if err := r.appendOutboxEvents(ctx, aggregate.PendingOutboxEvents()...); err != nil {
		return stackErr.Error(err)
	}
	aggregate.MarkPersisted()
	return nil
}

func (r *providerPaymentRepoImpl) GetByTransactionID(ctx context.Context, transactionID string) (*paymentaggregate.PaymentIntentAggregate, error) {
	var paymentIntent model.ProviderPaymentIntentModel
	if err := r.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&paymentIntent).Error; err != nil {
		return nil, mapError(err)
	}
	return toProviderPaymentAggregate(&paymentIntent)
}

func (r *providerPaymentRepoImpl) GetByExternalRef(ctx context.Context, provider, externalRef string) (*paymentaggregate.PaymentIntentAggregate, error) {
	var paymentIntent model.ProviderPaymentIntentModel
	if err := r.db.WithContext(ctx).
		Where("provider = ? AND external_ref = ?", provider, externalRef).
		First(&paymentIntent).Error; err != nil {
		return nil, mapError(err)
	}
	return toProviderPaymentAggregate(&paymentIntent)
}

func (r *providerPaymentRepoImpl) createIntent(ctx context.Context, intent *entity.PaymentIntent) error {
	err := r.db.WithContext(ctx).Create(&model.ProviderPaymentIntentModel{
		TransactionID:      intent.TransactionID,
		Provider:           intent.Provider,
		ExternalRef:        toStorageExternalRef(intent.Provider, intent.TransactionID, intent.ExternalRef),
		Amount:             intent.Amount,
		Currency:           intent.Currency,
		ClearingAccountKey: intent.ClearingAccountKey,
		CreditAccountID:    intent.CreditAccountID,
		Status:             intent.Status,
		CreatedAt:          intent.CreatedAt,
		UpdatedAt:          intent.UpdatedAt,
	}).Error
	return mapError(err)
}

func (r *providerPaymentRepoImpl) updateIntent(ctx context.Context, intent *entity.PaymentIntent) error {
	result := r.db.WithContext(ctx).
		Model(&model.ProviderPaymentIntentModel{}).
		Where("transaction_id = ?", intent.TransactionID).
		Updates(map[string]interface{}{
			"provider":             intent.Provider,
			"external_ref":         toStorageExternalRef(intent.Provider, intent.TransactionID, intent.ExternalRef),
			"amount":               intent.Amount,
			"currency":             intent.Currency,
			"clearing_account_key": intent.ClearingAccountKey,
			"credit_account_id":    intent.CreditAccountID,
			"status":               intent.Status,
			"updated_at":           intent.UpdatedAt,
		})
	if result.Error != nil {
		return mapError(result.Error)
	}
	if result.RowsAffected == 0 {
		return paymentrepos.ErrProviderPaymentNotFound
	}
	return nil
}

func (r *providerPaymentRepoImpl) markProcessed(ctx context.Context, event *entity.ProcessedPaymentEvent) error {
	err := r.db.WithContext(ctx).Create(&model.ProcessedProviderPaymentEventModel{
		Provider:       event.Provider,
		IdempotencyKey: event.IdempotencyKey,
		TransactionID:  event.TransactionID,
		CreatedAt:      event.CreatedAt,
	}).Error
	if isOracleUniqueConstraintError(err) {
		return paymentrepos.ErrProviderPaymentDuplicateProcessed
	}
	return mapError(err)
}

func (r *providerPaymentRepoImpl) appendOutboxEvents(ctx context.Context, events ...eventpkg.Event) error {
	for _, evt := range events {
		if err := r.appendOutboxEvent(ctx, evt); err != nil {
			return stackErr.Error(err)
		}
	}
	return nil
}

func (r *providerPaymentRepoImpl) appendOutboxEvent(ctx context.Context, evt eventpkg.Event) error {
	data, err := r.serializer.Marshal(evt.EventData)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal event data failed: %v", err))
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return stackErr.Error(r.db.WithContext(ctx).Create(&model.PaymentOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}).Error)
}

func toProviderPaymentAggregate(modelIntent *model.ProviderPaymentIntentModel) (*paymentaggregate.PaymentIntentAggregate, error) {
	externalRef := ""
	if modelIntent.ExternalRef != nil {
		externalRef = fromStorageExternalRef(*modelIntent.ExternalRef)
	}
	intent := normalizeProviderPaymentIntent(&entity.PaymentIntent{
		TransactionID:      modelIntent.TransactionID,
		Provider:           modelIntent.Provider,
		ExternalRef:        externalRef,
		Amount:             modelIntent.Amount,
		Currency:           modelIntent.Currency,
		ClearingAccountKey: modelIntent.ClearingAccountKey,
		CreditAccountID:    modelIntent.CreditAccountID,
		Status:             modelIntent.Status,
		CreatedAt:          modelIntent.CreatedAt,
		UpdatedAt:          modelIntent.UpdatedAt,
	})
	return paymentaggregate.RestorePaymentIntentAggregate(intent)
}

func aggregateSnapshot(aggregate *paymentaggregate.PaymentIntentAggregate) *entity.PaymentIntent {
	if aggregate == nil {
		return nil
	}
	return aggregate.Snapshot()
}

func toStorageExternalRef(provider, transactionID, externalRef string) *string {
	if value := strings.TrimSpace(externalRef); value != "" {
		return &value
	}

	placeholder := pendingExternalRefPrefix + strings.ToLower(strings.TrimSpace(provider)) + ":" + strings.TrimSpace(transactionID)
	return &placeholder
}

func fromStorageExternalRef(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, pendingExternalRefPrefix) {
		return ""
	}
	return value
}

func normalizeProviderPaymentIntent(intent *entity.PaymentIntent) *entity.PaymentIntent {
	if intent == nil {
		return nil
	}
	intent.Provider = strings.ToLower(strings.TrimSpace(intent.Provider))
	intent.TransactionID = strings.TrimSpace(intent.TransactionID)
	intent.ExternalRef = strings.TrimSpace(intent.ExternalRef)
	intent.Currency = strings.ToUpper(strings.TrimSpace(intent.Currency))
	intent.ClearingAccountKey = strings.TrimSpace(intent.ClearingAccountKey)
	intent.CreditAccountID = strings.TrimSpace(intent.CreditAccountID)
	intent.Status = entity.NormalizePaymentStatusOrPending(intent.Status)
	intent.CreatedAt = intent.CreatedAt.UTC()
	intent.UpdatedAt = intent.UpdatedAt.UTC()
	return intent
}
