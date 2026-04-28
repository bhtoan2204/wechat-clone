package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	paymentaggregate "wechat-clone/core/modules/payment/domain/aggregate"
	"wechat-clone/core/modules/payment/domain/entity"
	paymentrepos "wechat-clone/core/modules/payment/domain/repos"
	"wechat-clone/core/modules/payment/infra/persistent/model"
	shareddb "wechat-clone/core/shared/infra/db"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

const pendingExternalRefPrefix = "__pending__:"

type providerPaymentRepoImpl struct {
	db              *gorm.DB
	outboxPublisher eventpkg.Publisher
}

type paymentOutboxEventStore struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func newProviderPaymentRepoImpl(db *gorm.DB) paymentrepos.ProviderPaymentRepository {
	return &providerPaymentRepoImpl{
		db: db,
		outboxPublisher: eventpkg.NewPublisher(&paymentOutboxEventStore{
			db:         db,
			serializer: eventpkg.NewSerializer(),
		}),
	}
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
	if aggregate.Root().BaseVersion() == 0 {
		if err := r.createIntent(ctx, intent, aggregate.Version()); err != nil {
			return stackErr.Error(err)
		}
	} else if err := r.updateIntent(ctx, intent, aggregate.Version()); err != nil {
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

func (r *providerPaymentRepoImpl) ListPendingWithdrawals(ctx context.Context, limit int) ([]*paymentaggregate.PaymentIntentAggregate, error) {
	if limit <= 0 {
		limit = 20
	}

	var paymentIntents []model.ProviderPaymentIntentModel
	if err := r.db.WithContext(ctx).
		Where("workflow = ? AND status = ?", entity.PaymentWorkflowWithdrawal, entity.PaymentStatusCreating).
		Order("created_at ASC").
		Limit(limit).
		Find(&paymentIntents).Error; err != nil {
		return nil, mapError(err)
	}

	items := make([]*paymentaggregate.PaymentIntentAggregate, 0, len(paymentIntents))
	for idx := range paymentIntents {
		agg, err := toProviderPaymentAggregate(&paymentIntents[idx])
		if err != nil {
			return nil, stackErr.Error(err)
		}
		items = append(items, agg)
	}

	return items, nil
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

func (r *providerPaymentRepoImpl) createIntent(ctx context.Context, intent *entity.PaymentIntent, version int) error {
	err := r.db.WithContext(ctx).Create(&model.ProviderPaymentIntentModel{
		TransactionID:        intent.TransactionID,
		Workflow:             intent.Workflow,
		Provider:             intent.Provider,
		ExternalRef:          toStorageExternalRef(intent.Provider, intent.TransactionID, intent.ExternalRef),
		DestinationAccountID: toNullableString(intent.DestinationAccountID),
		Amount:               intent.Amount,
		FeeAmount:            intent.FeeAmount,
		ProviderAmount:       intent.ProviderAmount,
		Currency:             intent.Currency,
		ClearingAccountKey:   intent.ClearingAccountKey,
		DebitAccountID:       toNullableString(intent.DebitAccountID),
		CreditAccountID:      toNullableString(intent.CreditAccountID),
		Status:               intent.Status,
		Version:              version,
		CreatedAt:            intent.CreatedAt,
		UpdatedAt:            intent.UpdatedAt,
	}).Error
	return mapError(err)
}

func (r *providerPaymentRepoImpl) updateIntent(ctx context.Context, intent *entity.PaymentIntent, version int) error {
	result := r.db.WithContext(ctx).
		Model(&model.ProviderPaymentIntentModel{}).
		Where("transaction_id = ?", intent.TransactionID).
		Updates(map[string]interface{}{
			"workflow":               intent.Workflow,
			"provider":               intent.Provider,
			"external_ref":           toStorageExternalRef(intent.Provider, intent.TransactionID, intent.ExternalRef),
			"destination_account_id": toNullableString(intent.DestinationAccountID),
			"amount":                 intent.Amount,
			"fee_amount":             intent.FeeAmount,
			"provider_amount":        intent.ProviderAmount,
			"currency":               intent.Currency,
			"clearing_account_key":   intent.ClearingAccountKey,
			"debit_account_id":       toNullableString(intent.DebitAccountID),
			"credit_account_id":      toNullableString(intent.CreditAccountID),
			"status":                 intent.Status,
			"version":                version,
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
	if shareddb.IsUniqueConstraintError(err) {
		return paymentrepos.ErrProviderPaymentDuplicateProcessed
	}
	return mapError(err)
}

func (r *providerPaymentRepoImpl) appendOutboxEvents(ctx context.Context, events ...eventpkg.Event) error {
	if len(events) == 0 {
		return nil
	}
	if r == nil || r.outboxPublisher == nil {
		return stackErr.Error(eventpkg.ErrEventStoreNil)
	}
	return stackErr.Error(r.outboxPublisher.Publish(ctx, events...))
}

func (s *paymentOutboxEventStore) Append(ctx context.Context, evt eventpkg.Event) error {
	if s == nil || s.db == nil {
		return stackErr.Error(eventpkg.ErrEventStoreNil)
	}

	serializer := s.serializer
	if serializer == nil {
		serializer = eventpkg.NewSerializer()
	}
	data, err := serializer.Marshal(evt.EventData)
	if err != nil {
		return stackErr.Error(fmt.Errorf("marshal payment outbox event data failed: %w", err))
	}

	createdAt := time.Now().UTC()
	if evt.CreatedAt > 0 {
		createdAt = time.Unix(evt.CreatedAt, 0).UTC()
	}

	return stackErr.Error(s.db.WithContext(ctx).Create(&model.PaymentOutboxEventModel{
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
		Workflow:             modelIntent.Workflow,
		TransactionID:        modelIntent.TransactionID,
		Provider:             modelIntent.Provider,
		ExternalRef:          externalRef,
		DestinationAccountID: fromNullableString(modelIntent.DestinationAccountID),
		Amount:               modelIntent.Amount,
		FeeAmount:            modelIntent.FeeAmount,
		ProviderAmount:       modelIntent.ProviderAmount,
		Currency:             modelIntent.Currency,
		ClearingAccountKey:   modelIntent.ClearingAccountKey,
		DebitAccountID:       fromNullableString(modelIntent.DebitAccountID),
		CreditAccountID:      fromNullableString(modelIntent.CreditAccountID),
		Status:               modelIntent.Status,
		CreatedAt:            modelIntent.CreatedAt,
		UpdatedAt:            modelIntent.UpdatedAt,
	})
	return paymentaggregate.RestorePaymentIntentAggregateWithVersion(intent, modelIntent.Version)
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
	intent.Workflow = entity.NormalizePaymentWorkflow(intent.Workflow)
	if intent.Workflow == "" {
		intent.Workflow = entity.PaymentWorkflowTopUp
	}
	intent.Provider = strings.ToLower(strings.TrimSpace(intent.Provider))
	intent.TransactionID = strings.TrimSpace(intent.TransactionID)
	intent.ExternalRef = strings.TrimSpace(intent.ExternalRef)
	intent.DestinationAccountID = strings.TrimSpace(intent.DestinationAccountID)
	intent.Currency = strings.ToUpper(strings.TrimSpace(intent.Currency))
	intent.ClearingAccountKey = strings.TrimSpace(intent.ClearingAccountKey)
	intent.DebitAccountID = strings.TrimSpace(intent.DebitAccountID)
	intent.CreditAccountID = strings.TrimSpace(intent.CreditAccountID)
	if intent.ProviderAmount <= 0 {
		switch intent.Workflow {
		case entity.PaymentWorkflowWithdrawal:
			intent.ProviderAmount = intent.Amount
		default:
			intent.ProviderAmount = intent.Amount + intent.FeeAmount
		}
	}
	intent.Status = entity.NormalizePaymentStatusOrPending(intent.Status)
	intent.CreatedAt = intent.CreatedAt.UTC()
	intent.UpdatedAt = intent.UpdatedAt.UTC()
	return intent
}

func toNullableString(value string) *string {
	if value = strings.TrimSpace(value); value == "" {
		return nil
	}
	return &value
}

func fromNullableString(value *string) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(*value)
}
