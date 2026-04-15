package command

import (
	"context"

	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	ledgerout "go-socket/core/modules/ledger/application/dto/out"
	ledgerservice "go-socket/core/modules/ledger/application/service"
	"go-socket/core/shared/pkg/cqrs"
)

type createTransactionHandler struct {
	service *ledgerservice.LedgerService
}

func NewCreateTransactionHandler(service *ledgerservice.LedgerService) cqrs.Handler[*ledgerin.CreateTransactionRequest, *ledgerout.TransactionResponse] {
	return &createTransactionHandler{service: service}
}

func (h *createTransactionHandler) Handle(ctx context.Context, req *ledgerin.CreateTransactionRequest) (*ledgerout.TransactionResponse, error) {
	command := ledgerservice.CreateTransactionCommand{
		TransactionID: req.TransactionID,
		Currency:      req.Currency,
		Entries:       make([]ledgerservice.CreateTransactionEntryCommand, 0, len(req.Entries)),
	}
	for _, entry := range req.Entries {
		command.Entries = append(command.Entries, ledgerservice.CreateTransactionEntryCommand{
			AccountID: entry.AccountID,
			Amount:    entry.Amount,
		})
	}

	transaction, err := h.service.CreateTransaction(ctx, command)
	if err != nil {
		return nil, err
	}

	responseEntries := make([]ledgerout.LedgerEntryResponse, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		responseEntries = append(responseEntries, ledgerout.LedgerEntryResponse{
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
		Entries:       responseEntries,
	}, nil
}
