package query

import (
	"context"

	ledgerin "wechat-clone/core/modules/ledger/application/dto/in"
	ledgerout "wechat-clone/core/modules/ledger/application/dto/out"
	ledgerservice "wechat-clone/core/modules/ledger/application/service"
	"wechat-clone/core/shared/pkg/actorctx"
	"wechat-clone/core/shared/pkg/cqrs"
	"wechat-clone/core/shared/pkg/stackErr"
)

type getAccountBalanceHandler struct {
	service ledgerservice.LedgerQueryService
}

func NewGetAccountBalanceHandler(service ledgerservice.LedgerQueryService) cqrs.Handler[*ledgerin.GetAccountBalanceRequest, *ledgerout.AccountBalanceResponse] {
	return &getAccountBalanceHandler{service: service}
}

func (h *getAccountBalanceHandler) Handle(ctx context.Context, req *ledgerin.GetAccountBalanceRequest) (*ledgerout.AccountBalanceResponse, error) {
	accountID, err := actorctx.AccountIDFromContext(ctx)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return h.service.GetAccountBalance(ctx, accountID, req.Currency)
}
