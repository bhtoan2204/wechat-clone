package repository

import (
	"context"
	"strings"

	paymentaggregate "wechat-clone/core/modules/payment/domain/aggregate"
	"wechat-clone/core/modules/payment/domain/entity"
	"wechat-clone/core/modules/payment/domain/repos"
	"wechat-clone/core/modules/payment/infra/persistent/model"
	shareddb "wechat-clone/core/shared/infra/db"
	eventpkg "wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

const pendingExternalRefPrefix = "__pending__:"

type paymentIntentAggregateRepoImpl struct {
	db              *gorm.DB
	outboxPublisher eventpkg.Publisher
}

func NewPaymentIntentAggregateRepo(db *gorm.DB) repos.PaymentIntentAggregateRepo {
	return &paymentIntentAggregateRepoImpl{
		db: db,
		outboxPublisher: eventpkg.NewPublisher(&paymentOutboxEventStore{
			db:         db,
			serializer: eventpkg.NewSerializer(),
		}),
	}
}

func (r *paymentIntentAggregateRepoImpl) Save(ctx context.Context, agg *paymentaggregate.PaymentIntentAggregate) error {
	if agg == nil {
		return ErrPaymentIntentAggregateNotFound
	}

	intent := normalizePaymentIntent(agg.Snapshot())
	if intent == nil {
		return ErrPaymentIntentAggregateNotFound
	}

	for _, event := range agg.PendingProcessedEvents() {
		if err := r.insertProcessedEvent(ctx, event); err != nil {
			return stackErr.Error(err)
		}
	}

	modelIntent := toPaymentIntentModel(intent, agg.Version())

	if agg.Root().BaseVersion() == 0 {
		if err := mapError(r.db.WithContext(ctx).Create(modelIntent).Error); err != nil {
			return stackErr.Error(err)
		}
	} else {
		updates := paymentIntentUpdates(intent, agg.Version())

		result := r.db.WithContext(ctx).
			Model(&model.PaymentIntentModel{}).
			Where("transaction_id = ?", intent.TransactionID).
			Updates(updates)

		if result.Error != nil {
			return stackErr.Error(mapError(result.Error))
		}

		if result.RowsAffected == 0 {
			return ErrPaymentIntentAggregateNotFound
		}
	}

	if events := agg.PendingOutboxEvents(); len(events) > 0 {
		if r.outboxPublisher == nil {
			return stackErr.Error(eventpkg.ErrEventStoreNil)
		}

		if err := r.outboxPublisher.Publish(ctx, events...); err != nil {
			return stackErr.Error(err)
		}
	}

	agg.MarkPersisted()
	return nil
}

func (r *paymentIntentAggregateRepoImpl) GetByTransactionID(
	ctx context.Context,
	transactionID string,
) (*paymentaggregate.PaymentIntentAggregate, error) {
	var paymentIntent model.PaymentIntentModel

	if err := r.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&paymentIntent).Error; err != nil {
		return nil, mapError(err)
	}

	return toPaymentIntentAggregate(&paymentIntent)
}

func (r *paymentIntentAggregateRepoImpl) GetByExternalRef(
	ctx context.Context,
	provider string,
	externalRef string,
) (*paymentaggregate.PaymentIntentAggregate, error) {
	var paymentIntent model.PaymentIntentModel

	if err := r.db.WithContext(ctx).
		Where("provider = ? AND external_ref = ?", normalizeProvider(provider), strings.TrimSpace(externalRef)).
		First(&paymentIntent).Error; err != nil {
		return nil, mapError(err)
	}

	return toPaymentIntentAggregate(&paymentIntent)
}

func (r *paymentIntentAggregateRepoImpl) ListPendingWithdrawals(
	ctx context.Context,
	limit int,
) ([]*paymentaggregate.PaymentIntentAggregate, error) {
	if limit <= 0 {
		limit = 20
	}

	var paymentIntents []model.PaymentIntentModel

	if err := r.db.WithContext(ctx).
		Where("workflow = ? AND status = ?", entity.PaymentWorkflowWithdrawal, entity.PaymentStatusCreating).
		Order("created_at ASC").
		Limit(limit).
		Find(&paymentIntents).Error; err != nil {
		return nil, mapError(err)
	}

	items := make([]*paymentaggregate.PaymentIntentAggregate, 0, len(paymentIntents))

	for i := range paymentIntents {
		agg, err := toPaymentIntentAggregate(&paymentIntents[i])
		if err != nil {
			return nil, stackErr.Error(err)
		}

		items = append(items, agg)
	}

	return items, nil
}

func (r *paymentIntentAggregateRepoImpl) insertProcessedEvent(
	ctx context.Context,
	event *entity.ProcessedPaymentEvent,
) error {
	err := r.db.WithContext(ctx).Create(&model.ProcessedPaymentEventModel{
		Provider:       normalizeProvider(event.Provider),
		IdempotencyKey: strings.TrimSpace(event.IdempotencyKey),
		TransactionID:  strings.TrimSpace(event.TransactionID),
		CreatedAt:      event.CreatedAt.UTC(),
	}).Error

	if shareddb.IsUniqueConstraintError(err) {
		return ErrPaymentIntentAggregateDuplicateProcessed
	}

	return mapError(err)
}

func toPaymentIntentAggregate(
	modelIntent *model.PaymentIntentModel,
) (*paymentaggregate.PaymentIntentAggregate, error) {
	intent := normalizePaymentIntent(&entity.PaymentIntent{
		Workflow:             modelIntent.Workflow,
		TransactionID:        modelIntent.TransactionID,
		Provider:             modelIntent.Provider,
		ExternalRef:          externalRefFromStorage(modelIntent.ExternalRef),
		DestinationAccountID: stringFromDB(modelIntent.DestinationAccountID),
		Amount:               modelIntent.Amount,
		FeeAmount:            modelIntent.FeeAmount,
		ProviderAmount:       modelIntent.ProviderAmount,
		Currency:             modelIntent.Currency,
		ClearingAccountKey:   modelIntent.ClearingAccountKey,
		DebitAccountID:       stringFromDB(modelIntent.DebitAccountID),
		CreditAccountID:      stringFromDB(modelIntent.CreditAccountID),
		Status:               modelIntent.Status,
		CreatedAt:            modelIntent.CreatedAt,
		UpdatedAt:            modelIntent.UpdatedAt,
	})

	return paymentaggregate.RestorePaymentIntentAggregateWithVersion(intent, modelIntent.Version)
}

func toPaymentIntentModel(
	intent *entity.PaymentIntent,
	version int,
) *model.PaymentIntentModel {
	return &model.PaymentIntentModel{
		TransactionID:        intent.TransactionID,
		Workflow:             intent.Workflow,
		Provider:             intent.Provider,
		ExternalRef:          externalRefToStorage(intent.Provider, intent.TransactionID, intent.ExternalRef),
		DestinationAccountID: stringToDB(intent.DestinationAccountID),
		Amount:               intent.Amount,
		FeeAmount:            intent.FeeAmount,
		ProviderAmount:       intent.ProviderAmount,
		Currency:             intent.Currency,
		ClearingAccountKey:   intent.ClearingAccountKey,
		DebitAccountID:       stringToDB(intent.DebitAccountID),
		CreditAccountID:      stringToDB(intent.CreditAccountID),
		Status:               intent.Status,
		Version:              version,
		CreatedAt:            intent.CreatedAt,
		UpdatedAt:            intent.UpdatedAt,
	}
}

func paymentIntentUpdates(
	intent *entity.PaymentIntent,
	version int,
) map[string]interface{} {
	return map[string]interface{}{
		"workflow":               intent.Workflow,
		"provider":               intent.Provider,
		"external_ref":           externalRefToStorage(intent.Provider, intent.TransactionID, intent.ExternalRef),
		"destination_account_id": stringToDB(intent.DestinationAccountID),
		"amount":                 intent.Amount,
		"fee_amount":             intent.FeeAmount,
		"provider_amount":        intent.ProviderAmount,
		"currency":               intent.Currency,
		"clearing_account_key":   intent.ClearingAccountKey,
		"debit_account_id":       stringToDB(intent.DebitAccountID),
		"credit_account_id":      stringToDB(intent.CreditAccountID),
		"status":                 intent.Status,
		"version":                version,
	}
}

func normalizePaymentIntent(intent *entity.PaymentIntent) *entity.PaymentIntent {
	if intent == nil {
		return nil
	}

	intent.Workflow = entity.NormalizePaymentWorkflow(intent.Workflow)
	if intent.Workflow == "" {
		intent.Workflow = entity.PaymentWorkflowTopUp
	}

	intent.Provider = normalizeProvider(intent.Provider)
	intent.TransactionID = strings.TrimSpace(intent.TransactionID)
	intent.ExternalRef = strings.TrimSpace(intent.ExternalRef)
	intent.DestinationAccountID = strings.TrimSpace(intent.DestinationAccountID)
	intent.Currency = strings.ToUpper(strings.TrimSpace(intent.Currency))
	intent.ClearingAccountKey = strings.TrimSpace(intent.ClearingAccountKey)
	intent.DebitAccountID = strings.TrimSpace(intent.DebitAccountID)
	intent.CreditAccountID = strings.TrimSpace(intent.CreditAccountID)

	if intent.ProviderAmount <= 0 {
		if intent.Workflow == entity.PaymentWorkflowWithdrawal {
			intent.ProviderAmount = intent.Amount
		} else {
			intent.ProviderAmount = intent.Amount + intent.FeeAmount
		}
	}

	intent.Status = entity.NormalizePaymentStatusOrPending(intent.Status)
	intent.CreatedAt = intent.CreatedAt.UTC()
	intent.UpdatedAt = intent.UpdatedAt.UTC()

	return intent
}

func normalizeProvider(provider string) string {
	return strings.ToLower(strings.TrimSpace(provider))
}

func externalRefToStorage(provider, transactionID, externalRef string) *string {
	value := strings.TrimSpace(externalRef)
	if value != "" {
		return &value
	}

	placeholder := pendingExternalRefPrefix +
		normalizeProvider(provider) +
		":" +
		strings.TrimSpace(transactionID)

	return &placeholder
}

func externalRefFromStorage(value *string) string {
	if value == nil {
		return ""
	}

	externalRef := strings.TrimSpace(*value)
	if strings.HasPrefix(externalRef, pendingExternalRefPrefix) {
		return ""
	}

	return externalRef
}

func stringToDB(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}

	return &value
}

func stringFromDB(value *string) string {
	if value == nil {
		return ""
	}

	return strings.TrimSpace(*value)
}
