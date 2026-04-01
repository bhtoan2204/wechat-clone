package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-socket/core/modules/payment/domain/aggregate"
	"go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/domain/types"
	"go-socket/core/modules/payment/infra/persistent/model"
	eventpkg "go-socket/core/shared/pkg/event"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type paymentProjectionRepoImpl struct {
	db         *gorm.DB
	serializer eventpkg.Serializer
}

func NewPaymentProjectionRepoImpl(db *gorm.DB) repos.PaymentProjectionRepository {
	return &paymentProjectionRepoImpl{
		db:         db,
		serializer: newPaymentBalanceSerializer(),
	}
}

func (p *paymentProjectionRepoImpl) ProjectTransaction(ctx context.Context, eventID, transactionID, accountID string, amount, balanceDelta int64, transactionType types.TransactionType, createdAt time.Time) error {
	inserted, err := p.insertProjectedTransaction(ctx, eventID, transactionID, accountID, amount, transactionType, createdAt)
	if err != nil {
		return err
	}
	if !inserted {
		return nil
	}

	return p.applyBalanceDelta(ctx, accountID, balanceDelta, createdAt)
}

func (p *paymentProjectionRepoImpl) RebuildProjection(ctx context.Context, accountID string, mode repos.ProjectionRebuildMode) (*repos.ProjectionRebuildResult, error) {
	switch mode {
	case repos.ProjectionRebuildModeFull:
		return p.rebuildFullProjection(ctx, accountID)
	case repos.ProjectionRebuildModeSnapshot:
		return p.rebuildProjectionFromSnapshot(ctx, accountID)
	default:
		return nil, fmt.Errorf("unsupported projection rebuild mode: %s", mode)
	}
}

func (p *paymentProjectionRepoImpl) rebuildFullProjection(ctx context.Context, accountID string) (*repos.ProjectionRebuildResult, error) {
	aggregateIDs, err := p.listAggregateIDs(ctx, accountID)
	if err != nil {
		return nil, stackerr.Error(err)
	}
	if err := p.clearProjection(ctx, accountID, true); err != nil {
		return nil, stackerr.Error(err)
	}

	result := &repos.ProjectionRebuildResult{}
	for _, aggregateID := range aggregateIDs {
		eventModels, err := p.loadEventModels(ctx, aggregateID, 0)
		if err != nil {
			return nil, stackerr.Error(err)
		}
		if len(eventModels) == 0 {
			continue
		}

		result.Accounts++
		balanceTouched := false
		for _, eventModel := range eventModels {
			applied, err := p.replayPaymentEvent(ctx, eventModel, true)
			if err != nil {
				return nil, stackerr.Error(err)
			}
			if !applied {
				continue
			}
			result.EventsReplayed++
			result.TransactionsRebuilt++
			balanceTouched = true
		}
		if balanceTouched {
			result.BalancesRebuilt++
		}
	}

	return result, nil
}

func (p *paymentProjectionRepoImpl) rebuildProjectionFromSnapshot(ctx context.Context, accountID string) (*repos.ProjectionRebuildResult, error) {
	aggregateIDs, err := p.listAggregateIDs(ctx, accountID)
	if err != nil {
		return nil, stackerr.Error(err)
	}
	if err := p.clearProjection(ctx, accountID, false); err != nil {
		return nil, stackerr.Error(err)
	}

	result := &repos.ProjectionRebuildResult{}
	for _, aggregateID := range aggregateIDs {
		snapshot, err := p.loadLatestSnapshot(ctx, aggregateID)
		if err != nil {
			return nil, stackerr.Error(err)
		}

		startVersion := 0
		balanceTouched := false
		if snapshot != nil {
			if err := p.restoreBalanceProjectionFromSnapshot(ctx, *snapshot); err != nil {
				return nil, stackerr.Error(err)
			}
			startVersion = snapshot.Version
			balanceTouched = true
		}

		eventModels, err := p.loadEventModels(ctx, aggregateID, startVersion)
		if err != nil {
			return nil, stackerr.Error(err)
		}
		if len(eventModels) == 0 && !balanceTouched {
			continue
		}

		result.Accounts++
		for _, eventModel := range eventModels {
			applied, err := p.replayPaymentEvent(ctx, eventModel, false)
			if err != nil {
				return nil, stackerr.Error(err)
			}
			if !applied {
				continue
			}
			result.EventsReplayed++
			balanceTouched = true
		}
		if balanceTouched {
			result.BalancesRebuilt++
		}
	}

	return result, nil
}

func (p *paymentProjectionRepoImpl) insertProjectedTransaction(ctx context.Context, eventID, transactionID, accountID string, amount int64, transactionType types.TransactionType, createdAt time.Time) (bool, error) {
	if err := p.db.WithContext(ctx).Create(&model.PaymentTransactionModel{
		ID:        transactionID,
		AccountID: accountID,
		EventID:   eventID,
		Amount:    amount,
		Type:      transactionType.String(),
		CreatedAt: createdAt,
	}).Error; err != nil {
		if isDuplicatePaymentProjectionError(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func (p *paymentProjectionRepoImpl) applyBalanceDelta(ctx context.Context, accountID string, balanceDelta int64, createdAt time.Time) error {
	result := p.db.WithContext(ctx).
		Model(&model.BalanceModel{}).
		Where("account_id = ?", accountID).
		Update("amount", gorm.Expr("amount + ?", balanceDelta))
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}

	if err := p.db.WithContext(ctx).Create(&model.BalanceModel{
		ID:        accountID,
		AccountID: accountID,
		Amount:    balanceDelta,
		CreatedAt: createdAt,
	}).Error; err != nil {
		if isDuplicatePaymentProjectionError(err) {
			retry := p.db.WithContext(ctx).
				Model(&model.BalanceModel{}).
				Where("account_id = ?", accountID).
				Update("amount", gorm.Expr("amount + ?", balanceDelta))
			return retry.Error
		}
		return err
	}

	return nil
}

func (p *paymentProjectionRepoImpl) setBalanceAmount(ctx context.Context, accountID string, amount int64, createdAt time.Time) error {
	result := p.db.WithContext(ctx).
		Model(&model.BalanceModel{}).
		Where("account_id = ?", accountID).
		Update("amount", amount)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected > 0 {
		return nil
	}

	if err := p.db.WithContext(ctx).Create(&model.BalanceModel{
		ID:        accountID,
		AccountID: accountID,
		Amount:    amount,
		CreatedAt: createdAt,
	}).Error; err != nil {
		if isDuplicatePaymentProjectionError(err) {
			retry := p.db.WithContext(ctx).
				Model(&model.BalanceModel{}).
				Where("account_id = ?", accountID).
				Update("amount", amount)
			return retry.Error
		}
		return err
	}

	return nil
}

func (p *paymentProjectionRepoImpl) replayPaymentEvent(ctx context.Context, eventModel model.PaymentEventModel, projectTransactions bool) (bool, error) {
	payload, err := p.decodePaymentEvent(eventModel)
	if err != nil {
		return false, err
	}

	switch eventData := payload.(type) {
	case *aggregate.EventPaymentTransactionDeposited:
		if projectTransactions {
			return true, p.ProjectTransaction(ctx, eventModel.ID, eventData.PaymentTransactionID, eventModel.AggregateID, eventData.PaymentTransactionAmount, eventData.PaymentTransactionAmount, types.TransactionTypeDeposited, eventData.PaymentTransactionCreatedAt)
		}
		return true, p.applyBalanceDelta(ctx, eventModel.AggregateID, eventData.PaymentTransactionAmount, eventData.PaymentTransactionCreatedAt)
	case *aggregate.EventPaymentTransactionWithdrawn:
		if projectTransactions {
			return true, p.ProjectTransaction(ctx, eventModel.ID, eventData.PaymentTransactionID, eventModel.AggregateID, eventData.PaymentTransactionAmount, -eventData.PaymentTransactionAmount, types.TransactionTypeWithdrawn, eventData.PaymentTransactionCreatedAt)
		}
		return true, p.applyBalanceDelta(ctx, eventModel.AggregateID, -eventData.PaymentTransactionAmount, eventData.PaymentTransactionCreatedAt)
	case *aggregate.EventPaymentTransactionTransferred:
		if projectTransactions {
			return true, p.ProjectTransaction(ctx, eventModel.ID, eventData.PaymentTransactionID, eventModel.AggregateID, eventData.PaymentTransactionAmount, -eventData.PaymentTransactionAmount, types.TransactionTypeTransferred, eventData.PaymentTransactionCreatedAt)
		}
		return true, p.applyBalanceDelta(ctx, eventModel.AggregateID, -eventData.PaymentTransactionAmount, eventData.PaymentTransactionCreatedAt)
	case *aggregate.EventPaymentTransactionReceived:
		if projectTransactions {
			return true, p.ProjectTransaction(ctx, eventModel.ID, eventData.PaymentTransactionID, eventModel.AggregateID, eventData.PaymentTransactionAmount, eventData.PaymentTransactionAmount, types.TransactionTypeReceived, eventData.PaymentTransactionCreatedAt)
		}
		return true, p.applyBalanceDelta(ctx, eventModel.AggregateID, eventData.PaymentTransactionAmount, eventData.PaymentTransactionCreatedAt)
	default:
		return false, fmt.Errorf("unsupported payment event for projection rebuild: %s", eventModel.EventName)
	}
}

func (p *paymentProjectionRepoImpl) decodePaymentEvent(eventModel model.PaymentEventModel) (interface{}, error) {
	payloadFactory, ok := p.serializer.Type(eventModel.AggregateType, eventModel.EventName)
	if !ok {
		return nil, fmt.Errorf("unsupported payment event: aggregate_type=%s event_name=%s", eventModel.AggregateType, eventModel.EventName)
	}

	payload := clonePaymentPayload(payloadFactory())
	if payload == nil {
		return nil, fmt.Errorf("payment event payload prototype is nil")
	}
	if err := p.serializer.Unmarshal([]byte(eventModel.EventData), payload); err != nil {
		return nil, fmt.Errorf("unmarshal payment event payload failed: %w", err)
	}

	return payload, nil
}

func (p *paymentProjectionRepoImpl) restoreBalanceProjectionFromSnapshot(ctx context.Context, snapshot model.PaymentBalanceSnapshotModel) error {
	var agg aggregate.PaymentBalanceAggregate
	if err := p.serializer.Unmarshal([]byte(snapshot.State), &agg); err != nil {
		return fmt.Errorf("unmarshal payment snapshot state failed: %w", err)
	}

	createdAt := agg.CreatedAt
	if createdAt.IsZero() {
		createdAt = snapshot.CreatedAt
	}

	return p.setBalanceAmount(ctx, snapshot.AggregateID, agg.Balance, createdAt)
}

func (p *paymentProjectionRepoImpl) clearProjection(ctx context.Context, accountID string, includeTransactions bool) error {
	if includeTransactions {
		txQuery := p.db.WithContext(ctx)
		if accountID == "" {
			txQuery = txQuery.Session(&gorm.Session{AllowGlobalUpdate: true})
			if err := txQuery.Delete(&model.PaymentTransactionModel{}).Error; err != nil {
				return err
			}
		} else if err := txQuery.Where("account_id = ?", accountID).Delete(&model.PaymentTransactionModel{}).Error; err != nil {
			return err
		}
	}

	balanceQuery := p.db.WithContext(ctx)
	if accountID == "" {
		balanceQuery = balanceQuery.Session(&gorm.Session{AllowGlobalUpdate: true})
		return balanceQuery.Delete(&model.BalanceModel{}).Error
	}
	return balanceQuery.Where("account_id = ?", accountID).Delete(&model.BalanceModel{}).Error
}

func (p *paymentProjectionRepoImpl) listAggregateIDs(ctx context.Context, accountID string) ([]string, error) {
	query := p.db.WithContext(ctx).
		Model(&model.PaymentAggregateModel{}).
		Order("aggregate_id ASC")
	if accountID != "" {
		query = query.Where("aggregate_id = ?", accountID)
	}

	var aggregateIDs []string
	if err := query.Pluck("aggregate_id", &aggregateIDs).Error; err != nil {
		return nil, stackerr.Error(err)
	}

	return aggregateIDs, nil
}

func (p *paymentProjectionRepoImpl) loadEventModels(ctx context.Context, accountID string, afterVersion int) ([]model.PaymentEventModel, error) {
	query := p.db.WithContext(ctx).
		Where("aggregate_id = ?", accountID)
	if afterVersion > 0 {
		query = query.Where("version > ?", afterVersion)
	}

	var eventModels []model.PaymentEventModel
	if err := query.Order("version ASC").Find(&eventModels).Error; err != nil {
		return nil, stackerr.Error(err)
	}

	return eventModels, nil
}

func (p *paymentProjectionRepoImpl) loadLatestSnapshot(ctx context.Context, accountID string) (*model.PaymentBalanceSnapshotModel, error) {
	var snapshot model.PaymentBalanceSnapshotModel
	err := p.db.WithContext(ctx).
		Where("aggregate_id = ?", accountID).
		Order("version DESC").
		First(&snapshot).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, stackerr.Error(err)
	}

	return &snapshot, nil
}

func isDuplicatePaymentProjectionError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, gorm.ErrDuplicatedKey) || strings.Contains(err.Error(), "ORA-00001")
}
