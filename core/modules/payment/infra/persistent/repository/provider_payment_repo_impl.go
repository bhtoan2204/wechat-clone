package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

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

func (r *providerPaymentRepoImpl) CreatePaymentIntent(ctx context.Context, intent *entity.PaymentIntent, createdEvent eventpkg.Event) error {
	if err := r.CreateIntent(ctx, intent); err != nil {
		return stackErr.Error(err)
	}
	return stackErr.Error(r.appendOutboxEvents(ctx, createdEvent))
}

func (r *providerPaymentRepoImpl) SavePaymentIntent(ctx context.Context, intent *entity.PaymentIntent, outboxEvents ...eventpkg.Event) error {
	if intent == nil {
		return paymentrepos.ErrProviderPaymentNotFound
	}
	intent = normalizeProviderPaymentIntent(intent)

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
		})
	if result.Error != nil {
		return mapError(result.Error)
	}
	if result.RowsAffected == 0 {
		return paymentrepos.ErrProviderPaymentNotFound
	}

	return r.appendOutboxEvents(ctx, outboxEvents...)
}

func (r *providerPaymentRepoImpl) FinalizeSuccessfulPayment(
	ctx context.Context,
	intent *entity.PaymentIntent,
	processedEvent *entity.ProcessedPaymentEvent,
	successEvent eventpkg.Event,
	outboxEvents ...eventpkg.Event,
) error {
	if err := r.MarkProcessed(ctx, processedEvent); err != nil {
		return stackErr.Error(err)
	}
	outboxEvents = append(outboxEvents, successEvent)
	return stackErr.Error(r.SavePaymentIntent(ctx, intent, outboxEvents...))
}

func (r *providerPaymentRepoImpl) CreateIntent(ctx context.Context, intent *entity.PaymentIntent) error {
	intent = normalizeProviderPaymentIntent(intent)
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

func (r *providerPaymentRepoImpl) GetIntentByTransactionID(ctx context.Context, transactionID string) (*entity.PaymentIntent, error) {
	var paymentIntent model.ProviderPaymentIntentModel
	if err := r.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&paymentIntent).Error; err != nil {
		return nil, mapError(err)
	}
	return toProviderPaymentIntentEntity(&paymentIntent), nil
}

func (r *providerPaymentRepoImpl) GetIntentByExternalRef(ctx context.Context, provider, externalRef string) (*entity.PaymentIntent, error) {
	var paymentIntent model.ProviderPaymentIntentModel
	if err := r.db.WithContext(ctx).
		Where("provider = ? AND external_ref = ?", provider, externalRef).
		First(&paymentIntent).Error; err != nil {
		return nil, mapError(err)
	}
	return toProviderPaymentIntentEntity(&paymentIntent), nil
}

func (r *providerPaymentRepoImpl) UpdateIntentProviderState(ctx context.Context, transactionID, externalRef, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	if externalRef != "" {
		updates["external_ref"] = externalRef
	}

	result := r.db.WithContext(ctx).
		Model(&model.ProviderPaymentIntentModel{}).
		Where("transaction_id = ?", transactionID).
		Updates(updates)
	if result.Error != nil {
		return mapError(result.Error)
	}
	if result.RowsAffected == 0 {
		return paymentrepos.ErrProviderPaymentNotFound
	}
	return nil
}

func (r *providerPaymentRepoImpl) UpdateIntentStatus(ctx context.Context, transactionID, status string) error {
	result := r.db.WithContext(ctx).
		Model(&model.ProviderPaymentIntentModel{}).
		Where("transaction_id = ?", transactionID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return mapError(result.Error)
	}
	if result.RowsAffected == 0 {
		return paymentrepos.ErrProviderPaymentNotFound
	}
	return nil
}

func (r *providerPaymentRepoImpl) IsProcessed(ctx context.Context, provider, idempotencyKey string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.ProcessedProviderPaymentEventModel{}).
		Where("provider = ? AND idempotency_key = ?", provider, idempotencyKey).
		Count(&count).Error; err != nil {
		return false, mapError(err)
	}
	return count > 0, nil
}

func (r *providerPaymentRepoImpl) MarkProcessed(ctx context.Context, event *entity.ProcessedPaymentEvent) error {
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

func (r *providerPaymentRepoImpl) AppendOutboxEvent(ctx context.Context, evt eventpkg.Event) error {
	return r.appendOutboxEvents(ctx, evt)
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

func toProviderPaymentIntentEntity(modelIntent *model.ProviderPaymentIntentModel) *entity.PaymentIntent {
	externalRef := ""
	if modelIntent.ExternalRef != nil {
		externalRef = fromStorageExternalRef(*modelIntent.ExternalRef)
	}
	intent := &entity.PaymentIntent{
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
	}
	return normalizeProviderPaymentIntent(intent)
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
	intent.CreditAccountID = strings.TrimSpace(intent.CreditAccountID)
	if strings.TrimSpace(intent.ClearingAccountKey) == "" && intent.Provider != "" {
		intent.ClearingAccountKey = fmt.Sprintf("provider:%s", intent.Provider)
	}
	return intent
}
