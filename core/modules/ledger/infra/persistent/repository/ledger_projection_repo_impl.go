package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ledgerprojection "wechat-clone/core/modules/ledger/application/projection"
	"wechat-clone/core/modules/ledger/domain/entity"
	"wechat-clone/core/modules/ledger/infra/persistent/model"
	"wechat-clone/core/shared/pkg/stackErr"

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

	return stackErr.Error(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		ledgerRepo := NewLedgerRepoImpl(tx)

		existing, err := ledgerRepo.GetTransaction(ctx, transaction.TransactionID)
		if err == nil {
			if matchesProjectedTransaction(existing, transaction) {
				return nil
			}
			return stackErr.Error(fmt.Errorf("existing ledger projection mismatch for transaction_id=%s", transaction.TransactionID))
		}
		if !errors.Is(err, ErrNotFound) {
			return stackErr.Error(err)
		}

		if err := insertProjectedTransaction(ctx, tx, transaction); err != nil {
			if !errors.Is(err, ErrDuplicate) {
				return stackErr.Error(err)
			}

			existing, loadErr := ledgerRepo.GetTransaction(ctx, transaction.TransactionID)
			if loadErr != nil {
				return stackErr.Error(loadErr)
			}
			if matchesProjectedTransaction(existing, transaction) {
				return nil
			}
			return stackErr.Error(fmt.Errorf("existing ledger projection mismatch for transaction_id=%s", transaction.TransactionID))
		}

		return nil
	}))
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
	if !strings.EqualFold(strings.TrimSpace(existing.Currency), strings.TrimSpace(projected.Currency)) {
		return false
	}
	if !existing.CreatedAt.UTC().Equal(projected.CreatedAt.UTC()) {
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
		if !strings.EqualFold(strings.TrimSpace(existingEntry.Currency), strings.TrimSpace(projectedEntry.Currency)) {
			return false
		}
		if existingEntry.Amount != projectedEntry.Amount {
			return false
		}
		if !existingEntry.CreatedAt.UTC().Equal(projectedEntry.CreatedAt.UTC()) {
			return false
		}
	}

	return true
}

func insertProjectedTransaction(
	ctx context.Context,
	tx *gorm.DB,
	transaction *ledgerprojection.LedgerTransactionProjected,
) error {
	entryModels := make([]model.LedgerEntryModel, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		entryModels = append(entryModels, model.LedgerEntryModel{
			TransactionID: strings.TrimSpace(transaction.TransactionID),
			AccountID:     strings.TrimSpace(entry.AccountID),
			Currency:      strings.ToUpper(strings.TrimSpace(entry.Currency)),
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt.UTC(),
		})
	}

	if err := mapError(tx.WithContext(ctx).Create(&model.LedgerTransactionModel{
		TransactionID: strings.TrimSpace(transaction.TransactionID),
		Currency:      strings.ToUpper(strings.TrimSpace(transaction.Currency)),
		CreatedAt:     transaction.CreatedAt.UTC(),
	}).Error); err != nil {
		return stackErr.Error(err)
	}
	if len(entryModels) == 0 {
		return nil
	}
	if err := mapError(tx.WithContext(ctx).Create(&entryModels).Error); err != nil {
		return stackErr.Error(err)
	}

	return nil
}
