package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/stackErr"
)

type ledgerTransferPayload struct {
	TransactionID     string
	SenderAccountID   string
	ReceiverAccountID string
	Currency          string
	AmountMinor       int64
	CreatedAt         time.Time
}

func decodeLedgerAccountTransferPayload(ctx context.Context, raw json.RawMessage) (*ledgerTransferPayload, error) {
	payloadAny, err := decodeEventPayload(ctx, sharedevents.EventLedgerAccountTransferredToAccount, raw)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	payload, ok := payloadAny.(*sharedevents.LedgerAccountTransferredToAccountEvent)
	if !ok {
		return nil, stackErr.Error(fmt.Errorf("invalid payload type for event %s", sharedevents.EventLedgerAccountTransferredToAccount))
	}
	if payload == nil {
		return nil, stackErr.Error(fmt.Errorf("ledger transfer payload is required"))
	}
	if strings.TrimSpace(payload.TransactionID) == "" {
		return nil, stackErr.Error(fmt.Errorf("ledger transfer transaction_id is required"))
	}
	senderID, _ := ctx.Value(ledgerTransferSenderAccountIDKey{}).(string)
	senderID = strings.TrimSpace(senderID)
	if senderID == "" {
		return nil, stackErr.Error(fmt.Errorf("ledger transfer sender_account_id is required"))
	}
	receiverID := strings.TrimSpace(payload.ToAccountID)
	if senderID == "" || receiverID == "" {
		return nil, stackErr.Error(fmt.Errorf("ledger transfer payload account ids are required"))
	}
	if payload.Amount <= 0 {
		return nil, stackErr.Error(fmt.Errorf("ledger transfer amount must be positive"))
	}

	return &ledgerTransferPayload{
		TransactionID:     strings.TrimSpace(payload.TransactionID),
		SenderAccountID:   senderID,
		ReceiverAccountID: receiverID,
		Currency:          strings.ToUpper(strings.TrimSpace(payload.Currency)),
		AmountMinor:       payload.Amount,
		CreatedAt:         payload.BookedAt.UTC(),
	}, nil
}

type ledgerTransferSenderAccountIDKey struct{}

func transferMessageID(transactionID string) string {
	return "ledger-transfer:" + strings.TrimSpace(transactionID)
}

func formatTransferMessageBody(currency string, amountMinor int64) string {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	return currency + " " + strconv.FormatInt(amountMinor, 10)
}
