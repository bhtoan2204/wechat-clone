package query

import (
	"context"

	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	ledgerout "go-socket/core/modules/ledger/application/dto/out"
	ledgerservice "go-socket/core/modules/ledger/application/service"
	"go-socket/core/shared/pkg/cqrs"
)

type getTransactionHandler struct {
	service ledgerservice.LedgerQueryService
}

func NewGetTransactionHandler(service ledgerservice.LedgerQueryService) cqrs.Handler[*ledgerin.GetTransactionRequest, *ledgerout.TransactionResponse] {
	return &getTransactionHandler{service: service}
}

func (h *getTransactionHandler) Handle(ctx context.Context, req *ledgerin.GetTransactionRequest) (*ledgerout.TransactionResponse, error) {
	return h.service.GetTransaction(ctx, req.TransactionID)
}
