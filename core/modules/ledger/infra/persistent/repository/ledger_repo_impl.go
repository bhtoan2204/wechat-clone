package repository

import (
	"context"

	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	"go-socket/core/modules/ledger/infra/persistent/model"

	"gorm.io/gorm"
)

type ledgerRepoImpl struct {
	db *gorm.DB
}

func NewLedgerRepoImpl(db *gorm.DB) ledgerrepos.LedgerRepository {
	return &ledgerRepoImpl{db: db}
}

func (r *ledgerRepoImpl) CreateTransaction(ctx context.Context, transaction *entity.LedgerTransaction) error {
	err := r.db.WithContext(ctx).Create(&model.LedgerTransactionModel{
		TransactionID: transaction.TransactionID,
		CreatedAt:     transaction.CreatedAt,
	}).Error
	return mapError(err)
}

func (r *ledgerRepoImpl) InsertEntries(ctx context.Context, entries []*entity.LedgerEntry) error {
	models := make([]model.LedgerEntryModel, 0, len(entries))
	for _, entry := range entries {
		models = append(models, model.LedgerEntryModel{
			TransactionID: entry.TransactionID,
			AccountID:     entry.AccountID,
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt,
		})
	}
	return mapError(r.db.WithContext(ctx).Create(&models).Error)
}

func (r *ledgerRepoImpl) GetBalance(ctx context.Context, accountID string) (int64, error) {
	var balance int64
	err := r.db.WithContext(ctx).
		Model(&model.LedgerEntryModel{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("account_id = ?", accountID).
		Scan(&balance).Error
	return balance, mapError(err)
}

func (r *ledgerRepoImpl) GetTransaction(ctx context.Context, transactionID string) (*entity.LedgerTransaction, error) {
	var transactionModel model.LedgerTransactionModel
	if err := r.db.WithContext(ctx).
		Where("transaction_id = ?", transactionID).
		First(&transactionModel).Error; err != nil {
		return nil, mapError(err)
	}

	var entryModels []model.LedgerEntryModel
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
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt,
		})
	}

	return &entity.LedgerTransaction{
		TransactionID: transactionModel.TransactionID,
		CreatedAt:     transactionModel.CreatedAt,
		Entries:       entries,
	}, nil
}
