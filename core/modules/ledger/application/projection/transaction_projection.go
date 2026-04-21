package projection

import (
	"time"

	ledgeraggregate "wechat-clone/core/modules/ledger/domain/aggregate"
)

var ledgerTransactionProjectionEventNames = map[string]struct{}{
	ledgeraggregate.EventNameLedgerAccountDepositFromIntent:      {},
	ledgeraggregate.EventNameLedgerAccountWithdrawFromIntent:     {},
	ledgeraggregate.EventNameLedgerAccountDepositFromRefund:      {},
	ledgeraggregate.EventNameLedgerAccountWithdrawFromRefund:     {},
	ledgeraggregate.EventNameLedgerAccountDepositFromChargeback:  {},
	ledgeraggregate.EventNameLedgerAccountWithdrawFromChargeback: {},
	ledgeraggregate.EventNameLedgerAccountTransferredToAccount:   {},
	ledgeraggregate.EventNameLedgerAccountReceivedTransfer:       {},
}

type LedgerTransactionEntry struct {
	AccountID string    `json:"account_id"`
	Currency  string    `json:"currency"`
	Amount    int64     `json:"amount"`
	CreatedAt time.Time `json:"created_at"`
}

type LedgerTransactionProjected struct {
	TransactionID string                   `json:"transaction_id"`
	ReferenceType string                   `json:"reference_type"`
	ReferenceID   string                   `json:"reference_id"`
	Currency      string                   `json:"currency"`
	CreatedAt     time.Time                `json:"created_at"`
	Entries       []LedgerTransactionEntry `json:"entries"`
}

func IsLedgerTransactionProjectionEvent(name string) bool {
	_, ok := ledgerTransactionProjectionEventNames[name]
	return ok
}
