package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	appprojection "wechat-clone/core/modules/ledger/application/projection"
	"wechat-clone/core/modules/ledger/domain/entity"
	ledgerrepos "wechat-clone/core/modules/ledger/domain/repos"
	"wechat-clone/core/modules/ledger/infra/projection/views"
	shareddb "wechat-clone/core/shared/infra/db"
	"wechat-clone/core/shared/pkg/stackErr"

	"gorm.io/gorm"
)

var (
	ErrNotFound  = ledgerrepos.ErrNotFound
	ErrDuplicate = ledgerrepos.ErrDuplicate
)

type ledgerRepoImpl struct {
	db *gorm.DB
}

func NewLedgerRepoImpl(db *gorm.DB) appprojection.ReadRepository {
	return &ledgerRepoImpl{db: db}
}

func (r *ledgerRepoImpl) GetBalance(ctx context.Context, accountID, currency string) (int64, error) {
	var balance int64
	err := r.db.WithContext(ctx).
		Model(&views.LedgerEntryModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("account_id = ? AND currency = ?", accountID, currency).
		Scan(&balance).Error
	return balance, mapError(err)
}

func (r *ledgerRepoImpl) ProjectTransaction(ctx context.Context, transaction *appprojection.LedgerTransactionProjected) error {
	return stackErr.Error(r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := upsertProjectedTransactionHeader(ctx, tx, transaction); err != nil {
			return stackErr.Error(err)
		}
		if err := upsertProjectedTransactionEntries(ctx, tx, transaction); err != nil {
			return stackErr.Error(err)
		}
		return nil
	}))
}

func (r *ledgerRepoImpl) CountTransactions(ctx context.Context, accountID, currency string) (int64, error) {
	accountID = strings.TrimSpace(accountID)
	currency = strings.ToUpper(strings.TrimSpace(currency))

	query := r.listTransactionsBaseQuery(ctx, accountID, currency)

	var total int64
	err := query.Distinct("t.transaction_id").Count(&total).Error
	return total, mapError(err)
}

func (r *ledgerRepoImpl) GetTransaction(ctx context.Context, transactionID string) (*entity.LedgerTransaction, error) {
	var transactionViews views.LedgerTransactionModel
	if err := r.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&transactionViews).Error; err != nil {
		return nil, mapError(err)
	}

	var entryModels []views.LedgerEntryModel
	if err := r.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		Order("id ASC").
		Find(&entryModels).Error; err != nil {
		return nil, mapError(err)
	}

	entries := make([]*entity.LedgerEntry, 0, len(entryModels))
	for _, entryModel := range entryModels {
		entry := entryModel
		entries = append(entries, &entity.LedgerEntry{
			ID:            entry.ID,
			TransactionID: entry.TransactionID,
			AccountID:     entry.AccountID,
			Currency:      entry.Currency,
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt,
		})
	}

	return &entity.LedgerTransaction{
		TransactionID: transactionViews.TransactionID,
		Currency:      transactionViews.Currency,
		CreatedAt:     transactionViews.CreatedAt,
		Entries:       entries,
	}, nil
}

func (r *ledgerRepoImpl) ListTransactions(ctx context.Context, filter appprojection.ListTransactionsFilter) ([]*entity.LedgerTransaction, error) {
	query := r.listTransactionsBaseQuery(ctx, filter.AccountID, filter.Currency)

	if filter.CursorCreatedAt != nil && strings.TrimSpace(filter.CursorTransactionID) != "" {
		query = query.Where(
			"(t.created_at < ? OR (t.created_at = ? AND t.transaction_id < ?))",
			filter.CursorCreatedAt.UTC(),
			filter.CursorCreatedAt.UTC(),
			strings.TrimSpace(filter.CursorTransactionID),
		)
	}
	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}

	var transactionRows []views.LedgerTransactionListRow
	if err := query.
		Select("t.transaction_id, t.currency, t.created_at").
		Group("t.transaction_id, t.currency, t.created_at").
		Order("t.created_at DESC").
		Order("t.transaction_id DESC").
		Find(&transactionRows).Error; err != nil {
		return nil, mapError(err)
	}
	if len(transactionRows) == 0 {
		return []*entity.LedgerTransaction{}, nil
	}

	transactionIDs := make([]string, 0, len(transactionRows))
	transactionsByID := make(map[string]*entity.LedgerTransaction, len(transactionRows))
	for _, row := range transactionRows {
		transactionIDs = append(transactionIDs, row.TransactionID)
		transactionsByID[row.TransactionID] = &entity.LedgerTransaction{
			TransactionID: row.TransactionID,
			Currency:      row.Currency,
			CreatedAt:     row.CreatedAt,
			Entries:       make([]*entity.LedgerEntry, 0),
		}
	}

	var entryModels []views.LedgerEntryModel
	if err := r.db.WithContext(ctx).
		Where("transaction_id IN ?", transactionIDs).
		Order("transaction_id ASC").
		Order("id ASC").
		Find(&entryModels).Error; err != nil {
		return nil, mapError(err)
	}

	for _, entryModel := range entryModels {
		transaction := transactionsByID[entryModel.TransactionID]
		if transaction == nil {
			continue
		}
		entry := entryModel
		transaction.Entries = append(transaction.Entries, &entity.LedgerEntry{
			ID:            entry.ID,
			TransactionID: entry.TransactionID,
			AccountID:     entry.AccountID,
			Currency:      entry.Currency,
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt,
		})
	}

	transactions := make([]*entity.LedgerTransaction, 0, len(transactionRows))
	for _, row := range transactionRows {
		transactions = append(transactions, transactionsByID[row.TransactionID])
	}

	return transactions, nil
}

func (r *ledgerRepoImpl) listTransactionsBaseQuery(ctx context.Context, accountID, currency string) *gorm.DB {
	accountID = strings.TrimSpace(accountID)
	currency = strings.ToUpper(strings.TrimSpace(currency))

	query := r.db.WithContext(ctx).
		Table(views.LedgerTransactionModel{}.TableName()+" t").
		Joins("JOIN "+views.LedgerEntryModel{}.TableName()+" e ON e.transaction_id = t.transaction_id").
		Where("e.account_id = ?", accountID)

	if currency != "" {
		query = query.Where("t.currency = ?", currency)
	}

	return query
}

func mapError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return stackErr.Error(ErrNotFound)
	}

	if shareddb.IsUniqueConstraintError(err) {
		return stackErr.Error(ErrDuplicate)
	}
	return stackErr.Error(err)
}

func upsertProjectedTransactionHeader(
	ctx context.Context,
	tx *gorm.DB,
	transaction *appprojection.LedgerTransactionProjected,
) error {
	transactionID := strings.TrimSpace(transaction.TransactionID)
	projectedCreatedAt := normalizeProjectionTimestamp(transaction.CreatedAt)
	var existing views.LedgerTransactionModel
	err := tx.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&existing).Error
	if err == nil {
		if !strings.EqualFold(strings.TrimSpace(existing.Currency), strings.TrimSpace(transaction.Currency)) {
			return stackErr.Error(fmt.Errorf("existing ledger projection currency mismatch for transaction_id=%s", transactionID))
		}
		if !normalizeProjectionTimestamp(existing.CreatedAt).Equal(projectedCreatedAt) {
			return stackErr.Error(fmt.Errorf("existing ledger projection created_at mismatch for transaction_id=%s", transactionID))
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return stackErr.Error(mapError(err))
	}

	if err := mapError(tx.WithContext(ctx).Create(&views.LedgerTransactionModel{
		TransactionID: transactionID,
		Currency:      strings.ToUpper(strings.TrimSpace(transaction.Currency)),
		CreatedAt:     projectedCreatedAt,
	}).Error); err != nil {
		if !errors.Is(err, ErrDuplicate) {
			return stackErr.Error(err)
		}
	}

	return nil
}

func upsertProjectedTransactionEntries(
	ctx context.Context,
	tx *gorm.DB,
	transaction *appprojection.LedgerTransactionProjected,
) error {
	if len(transaction.Entries) == 0 {
		return nil
	}

	transactionID := strings.TrimSpace(transaction.TransactionID)
	var existingEntries []views.LedgerEntryModel
	if err := tx.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		Find(&existingEntries).Error; err != nil {
		return stackErr.Error(mapError(err))
	}

	for _, projectedEntry := range transaction.Entries {
		projectedEntry.CreatedAt = normalizeProjectionTimestamp(projectedEntry.CreatedAt)
		if hasMatchingProjectedEntry(existingEntries, transactionID, projectedEntry) {
			continue
		}
		if hasConflictingProjectedEntry(existingEntries, transactionID, projectedEntry) {
			return stackErr.Error(fmt.Errorf(
				"existing ledger projection entry mismatch for transaction_id=%s account_id=%s",
				transactionID,
				strings.TrimSpace(projectedEntry.AccountID),
			))
		}
		entryModel := views.LedgerEntryModel{
			TransactionID: transactionID,
			AccountID:     strings.TrimSpace(projectedEntry.AccountID),
			Currency:      strings.ToUpper(strings.TrimSpace(projectedEntry.Currency)),
			Amount:        projectedEntry.Amount,
			CreatedAt:     projectedEntry.CreatedAt,
		}
		if err := mapError(tx.WithContext(ctx).Create(&entryModel).Error); err != nil {
			return stackErr.Error(err)
		}
		existingEntries = append(existingEntries, entryModel)
	}

	return nil
}

func hasMatchingProjectedEntry(
	existingEntries []views.LedgerEntryModel,
	transactionID string,
	projectedEntry appprojection.LedgerTransactionEntry,
) bool {
	for _, existingEntry := range existingEntries {
		if strings.TrimSpace(existingEntry.TransactionID) != transactionID {
			continue
		}
		if strings.TrimSpace(existingEntry.AccountID) != strings.TrimSpace(projectedEntry.AccountID) {
			continue
		}
		if !strings.EqualFold(strings.TrimSpace(existingEntry.Currency), strings.TrimSpace(projectedEntry.Currency)) {
			continue
		}
		if existingEntry.Amount != projectedEntry.Amount {
			continue
		}
		if !normalizeProjectionTimestamp(existingEntry.CreatedAt).Equal(projectedEntry.CreatedAt) {
			continue
		}
		return true
	}
	return false
}

func hasConflictingProjectedEntry(
	existingEntries []views.LedgerEntryModel,
	transactionID string,
	projectedEntry appprojection.LedgerTransactionEntry,
) bool {
	for _, existingEntry := range existingEntries {
		if strings.TrimSpace(existingEntry.TransactionID) != transactionID {
			continue
		}
		if strings.TrimSpace(existingEntry.AccountID) != strings.TrimSpace(projectedEntry.AccountID) {
			continue
		}
		return !hasMatchingProjectedEntry([]views.LedgerEntryModel{existingEntry}, transactionID, projectedEntry)
	}
	return false
}

func normalizeProjectionTimestamp(value time.Time) time.Time {
	return value.UTC().Truncate(time.Microsecond)
}

// func (r *ledgerRepoImpl) GetTransaction(ctx context.Context, transactionID string) (*entity.LedgerTransaction, error) {
// 	var transactionModel views.LedgerTransactionModel
// 	if err := r.db.WithContext(ctx).
// 		Where("transaction_id = ?", transactionID).
// 		First(&transactionModel).Error; err != nil {
// 		return nil, mapError(err)
// 	}

// 	var entryModels []views.LedgerEntryModel
// 	if err := r.db.WithContext(ctx).
// 		Where("transaction_id = ?", transactionID).
// 		Order("id ASC").
// 		Find(&entryModels).Error; err != nil {
// 		return nil, mapError(err)
// 	}

// 	entries := make([]*entity.LedgerEntry, 0, len(entryModels))
// 	for _, entryModel := range entryModels {
// 		entry := entryModel
// 		entries = append(entries, &entity.LedgerEntry{
// 			ID:            entry.ID,
// 			TransactionID: entry.TransactionID,
// 			AccountID:     entry.AccountID,
// 			Currency:      entry.Currency,
// 			Amount:        entry.Amount,
// 			CreatedAt:     entry.CreatedAt,
// 		})
// 	}

// 	return &entity.LedgerTransaction{
// 		TransactionID: transactionModel.TransactionID,
// 		Currency:      transactionModel.Currency,
// 		CreatedAt:     transactionModel.CreatedAt,
// 		Entries:       entries,
// 	}, nil
// }
