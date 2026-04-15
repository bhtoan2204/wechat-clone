package service

import (
	ledgerout "go-socket/core/modules/ledger/application/dto/out"
	"go-socket/core/modules/ledger/domain/entity"
)

func toTransactionResponse(transaction *entity.LedgerTransaction) *ledgerout.TransactionResponse {
	entries := make([]ledgerout.LedgerEntryResponse, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		entries = append(entries, ledgerout.LedgerEntryResponse{
			ID:            entry.ID,
			TransactionID: entry.TransactionID,
			AccountID:     entry.AccountID,
			Currency:      entry.Currency,
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt,
		})
	}

	return &ledgerout.TransactionResponse{
		TransactionID: transaction.TransactionID,
		Currency:      transaction.Currency,
		CreatedAt:     transaction.CreatedAt,
		Entries:       entries,
	}
}
