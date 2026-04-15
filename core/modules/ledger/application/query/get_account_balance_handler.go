package query

import (
	"context"

	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	ledgerout "go-socket/core/modules/ledger/application/dto/out"
	ledgerservice "go-socket/core/modules/ledger/application/service"
	"go-socket/core/shared/pkg/cqrs"
)

type getAccountBalanceHandler struct {
	service ledgerservice.LedgerQueryService
}

func NewGetAccountBalanceHandler(service ledgerservice.LedgerQueryService) cqrs.Handler[*ledgerin.GetAccountBalanceRequest, *ledgerout.AccountBalanceResponse] {
	return &getAccountBalanceHandler{service: service}
}

func (h *getAccountBalanceHandler) Handle(ctx context.Context, req *ledgerin.GetAccountBalanceRequest) (*ledgerout.AccountBalanceResponse, error) {
	return h.service.GetAccountBalance(ctx, req.AccountID, req.Currency)
}
