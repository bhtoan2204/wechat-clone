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
	return h.service.CreateTransaction(ctx, req)
}
