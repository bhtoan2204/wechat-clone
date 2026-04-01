package paymentledger

import (
	"context"

	appCtx "go-socket/core/context"
	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	ledgerservice "go-socket/core/modules/ledger/application/service"
	ledgerassembly "go-socket/core/modules/ledger/assembly"
	paymentservice "go-socket/core/modules/payment/application/service"
)

const ServiceName = "payment.ledger_gateway"

type gateway struct {
	ledgerService *ledgerservice.LedgerService
}

func NewGateway(appContext *appCtx.AppContext) paymentservice.LedgerGateway {
	return &gateway{
		ledgerService: ledgerassembly.BuildService(appContext),
	}
}

func (g *gateway) PostTransaction(ctx context.Context, req paymentservice.LedgerPostingRequest) error {
	entries := make([]ledgerin.LedgerEntryInput, 0, len(req.Entries))
	for _, entry := range req.Entries {
		entries = append(entries, ledgerin.LedgerEntryInput{
			AccountID: entry.AccountID,
			Amount:    entry.Amount,
		})
	}

	_, err := g.ledgerService.CreateTransaction(ctx, &ledgerin.CreateTransactionRequest{
		TransactionID: req.TransactionID,
		Entries:       entries,
	})
	return err
}
