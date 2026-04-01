package repository

import (
	"context"
	"time"

	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	"go-socket/core/modules/ledger/infra/persistent/model"

	"gorm.io/gorm"
)

type paymentRepoImpl struct {
	db *gorm.DB
}

func NewPaymentRepoImpl(db *gorm.DB) ledgerrepos.PaymentRepository {
	return &paymentRepoImpl{db: db}
}

func (r *paymentRepoImpl) CreateIntent(ctx context.Context, intent *entity.PaymentIntent) error {
	err := r.db.WithContext(ctx).Create(&model.PaymentIntentModel{
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

func (r *paymentRepoImpl) GetIntentByTransactionID(ctx context.Context, transactionID string) (*entity.PaymentIntent, error) {
	var paymentIntent model.PaymentIntentModel
	if err := r.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&paymentIntent).Error; err != nil {
		return nil, mapError(err)
	}
	return toPaymentIntentEntity(&paymentIntent), nil
}

func (r *paymentRepoImpl) GetIntentByExternalRef(ctx context.Context, provider, externalRef string) (*entity.PaymentIntent, error) {
	var paymentIntent model.PaymentIntentModel
	if err := r.db.WithContext(ctx).
		Where("provider = ? AND external_ref = ?", provider, externalRef).
		First(&paymentIntent).Error; err != nil {
		return nil, mapError(err)
	}
	return toPaymentIntentEntity(&paymentIntent), nil
}

func (r *paymentRepoImpl) UpdateIntentProviderState(ctx context.Context, transactionID, externalRef, status string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now().UTC(),
	}
	if externalRef != "" {
		updates["external_ref"] = externalRef
	}

	result := r.db.WithContext(ctx).
		Model(&model.PaymentIntentModel{}).
		Where("transaction_id = ?", transactionID).
		Updates(updates)
	if result.Error != nil {
		return mapError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *paymentRepoImpl) UpdateIntentStatus(ctx context.Context, transactionID, status string) error {
	result := r.db.WithContext(ctx).
		Model(&model.PaymentIntentModel{}).
		Where("transaction_id = ?", transactionID).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now().UTC(),
		})
	if result.Error != nil {
		return mapError(result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *paymentRepoImpl) IsProcessed(ctx context.Context, provider, idempotencyKey string) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).
		Model(&model.ProcessedPaymentEventModel{}).
		Where("provider = ? AND idempotency_key = ?", provider, idempotencyKey).
		Count(&count).Error; err != nil {
		return false, mapError(err)
	}
	return count > 0, nil
}

func (r *paymentRepoImpl) MarkProcessed(ctx context.Context, event *entity.ProcessedPaymentEvent) error {
	err := r.db.WithContext(ctx).Create(&model.ProcessedPaymentEventModel{
		Provider:       event.Provider,
		IdempotencyKey: event.IdempotencyKey,
		TransactionID:  event.TransactionID,
		CreatedAt:      event.CreatedAt,
	}).Error
	return mapError(err)
}

func toPaymentIntentEntity(modelIntent *model.PaymentIntentModel) *entity.PaymentIntent {
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
