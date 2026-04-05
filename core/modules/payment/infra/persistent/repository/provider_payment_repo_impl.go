package repository

import (
	"context"
	"fmt"
	"time"

	"go-socket/core/modules/payment/domain/entity"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/infra/persistent/model"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

type providerPaymentRepoImpl struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func NewProviderPaymentRepoImpl(db *gorm.DB) paymentrepos.ProviderPaymentRepository {
	return &providerPaymentRepoImpl{
		db:         db,
		serializer: eventpkg.NewSerializer(),
	}
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

func (r *providerPaymentRepoImpl) AppendOutboxEvent(ctx context.Context, evt eventpkg.Event) error {
	data, err := r.serializer.Marshal(evt.EventData)
	if err != nil {
		return fmt.Errorf("marshal event data failed: %v", err)
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return r.db.WithContext(ctx).Create(&model.PaymentOutboxEventModel{
		AggregateID:   evt.AggregateID,
		AggregateType: evt.AggregateType,
		Version:       evt.Version,
		EventName:     evt.EventName,
		EventData:     string(data),
		Metadata:      "{}",
		CreatedAt:     createdAt,
	}).Error
}

func (r *providerPaymentRepoImpl) WithTransaction(ctx context.Context, fn func(paymentrepos.ProviderPaymentRepository) error) (err error) {
	log := logging.FromContext(ctx).Named("ProviderPaymentTransaction")
	tx := r.db.WithContext(ctx).Begin()
	if beginErr := tx.Error; beginErr != nil {
		log.Errorw("failed to begin transaction", zap.Error(beginErr))
		return beginErr
	}

	tr := NewProviderPaymentRepoImpl(tx)

	defer func() {
		if rec := recover(); rec != nil {
			_ = tx.Rollback().Error
			log.Errorw("panic -> rollback", zap.Any("panic", rec))
			panic(rec)
		}

		if err != nil {
			_ = tx.Rollback().Error
			log.Errorw("transaction rollback", zap.Error(err))
			return
		}

		if commitErr := tx.Commit().Error; commitErr != nil {
			log.Errorw("commit failed", zap.Error(commitErr))
			err = commitErr
			return
		}

		log.Info("transaction committed")
	}()

	err = fn(tr)
	return err
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
