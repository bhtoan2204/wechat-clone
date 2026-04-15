package repository

import (
	"context"
	"fmt"
	"strings"

	ledgerprojection "go-socket/core/modules/ledger/application/projection"
	"go-socket/core/modules/ledger/domain/entity"
	"go-socket/core/modules/ledger/infra/persistent/model"
	"go-socket/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

type ledgerProjectionRepoImpl struct {
	db *gorm.DB
}

func NewLedgerProjectionRepoImpl(db *gorm.DB) ledgerprojection.Projector {
	return &ledgerProjectionRepoImpl{db: db}
}

func (r *ledgerProjectionRepoImpl) ProjectTransaction(ctx context.Context, transaction *ledgerprojection.LedgerTransactionProjected) error {
	if transaction == nil {
		return stackErr.Error(fmt.Errorf("ledger transaction projection is nil"))
	}
	if strings.TrimSpace(transaction.TransactionID) == "" {
		return stackErr.Error(fmt.Errorf("ledger transaction projection id is required"))
	}

	existing, err := NewLedgerRepoImpl(r.db).GetTransaction(ctx, transaction.TransactionID)
	if err == nil {
		if matchesProjectedTransaction(existing, transaction) {
			return nil
		}
		return stackErr.Error(fmt.Errorf("existing ledger projection mismatch for transaction_id=%s", transaction.TransactionID))
	}
	if err != nil && err != ErrNotFound {
		return stackErr.Error(err)
	}

	entryModels := make([]model.LedgerEntryModel, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		entryModels = append(entryModels, model.LedgerEntryModel{
			TransactionID: transaction.TransactionID,
			AccountID:     strings.TrimSpace(entry.AccountID),
			Currency:      strings.ToUpper(strings.TrimSpace(entry.Currency)),
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt.UTC(),
		})
	}

	if err := mapError(r.db.WithContext(ctx).Create(&model.LedgerTransactionModel{
		TransactionID: transaction.TransactionID,
		Currency:      strings.ToUpper(strings.TrimSpace(transaction.Currency)),
		CreatedAt:     transaction.CreatedAt.UTC(),
	}).Error); err != nil {
		return stackErr.Error(err)
	}
	if err := mapError(r.db.WithContext(ctx).Create(&entryModels).Error); err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func matchesProjectedTransaction(
	existing *entity.LedgerTransaction,
	projected *ledgerprojection.LedgerTransactionProjected,
) bool {
	if existing == nil || projected == nil {
		return false
	}
	if strings.TrimSpace(existing.TransactionID) != strings.TrimSpace(projected.TransactionID) {
		return false
	}
	if strings.ToUpper(strings.TrimSpace(existing.Currency)) != strings.ToUpper(strings.TrimSpace(projected.Currency)) {
		return false
	}
	if len(existing.Entries) != len(projected.Entries) {
		return false
	}

	for idx, existingEntry := range existing.Entries {
		projectedEntry := projected.Entries[idx]
		if existingEntry == nil {
			return false
		}
		if strings.TrimSpace(existingEntry.AccountID) != strings.TrimSpace(projectedEntry.AccountID) {
			return false
		}
		if strings.ToUpper(strings.TrimSpace(existingEntry.Currency)) != strings.ToUpper(strings.TrimSpace(projectedEntry.Currency)) {
			return false
		}
		if existingEntry.Amount != projectedEntry.Amount {
			return false
		}
	}

	return true
}
