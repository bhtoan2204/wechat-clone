package repository

import (
	"context"
	"time"

	"go-socket/core/modules/payment/domain/entity"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/infra/persistent/model"

	"gorm.io/gorm"
)

type providerPaymentRepoImpl struct {
	db *gorm.DB
}

func NewProviderPaymentRepoImpl(db *gorm.DB) paymentrepos.ProviderPaymentRepository {
	return &providerPaymentRepoImpl{db: db}
}

func (r *providerPaymentRepoImpl) CreateIntent(ctx context.Context, intent *entity.PaymentIntent) error {
	err := r.db.WithContext(ctx).Create(&model.ProviderPaymentIntentModel{
		TransactionID:   intent.TransactionID,
		Provider:        intent.Provider,
		ExternalRef:     toNullableString(intent.ExternalRef),
		Amount:          intent.Amount,
		Currency:        intent.Currency,
		DebitAccountID:  intent.DebitAccountID,
		CreditAccountID: intent.CreditAccountID,
		Status:          intent.Status,
		CreatedAt:       intent.CreatedAt,
		UpdatedAt:       intent.UpdatedAt,
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
		"status":     status,
		"updated_at": time.Now().UTC(),
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

func toProviderPaymentIntentEntity(modelIntent *model.ProviderPaymentIntentModel) *entity.PaymentIntent {
	externalRef := ""
	if modelIntent.ExternalRef != nil {
		externalRef = *modelIntent.ExternalRef
	}
	return &entity.PaymentIntent{
		TransactionID:   modelIntent.TransactionID,
		Provider:        modelIntent.Provider,
		ExternalRef:     externalRef,
		Amount:          modelIntent.Amount,
		Currency:        modelIntent.Currency,
		DebitAccountID:  modelIntent.DebitAccountID,
		CreditAccountID: modelIntent.CreditAccountID,
		Status:          modelIntent.Status,
		CreatedAt:       modelIntent.CreatedAt,
		UpdatedAt:       modelIntent.UpdatedAt,
	}
}

func toNullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
