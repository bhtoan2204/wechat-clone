package aggregate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"go-socket/core/modules/ledger/domain/entity"
	"go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

var (
	ErrLedgerAccountAggregateRequired   = errors.New("ledger account aggregate is required")
	ErrLedgerAccountIDRequired          = errors.New("ledger account id is required")
	ErrLedgerAccountIDMismatch          = errors.New("ledger account id mismatch")
	ErrLedgerAccountTransactionRequired = errors.New("ledger transaction id is required")
	ErrLedgerAccountCurrencyRequired    = errors.New("ledger currency is required")
	ErrLedgerAccountAmountInvalid       = errors.New("ledger amount must be positive")
	ErrLedgerAccountBookedAtRequired    = errors.New("ledger booked_at is required")
	ErrLedgerAccountInsufficientFunds   = errors.New("ledger account has insufficient funds")
)

type LedgerAccountPosting struct {
	TransactionID         string    `json:"transaction_id"`
	ReferenceType         string    `json:"reference_type"`
	ReferenceID           string    `json:"reference_id"`
	CounterpartyAccountID string    `json:"counterparty_account_id"`
	Currency              string    `json:"currency"`
	AmountDelta           int64     `json:"amount_delta"`
	BookedAt              time.Time `json:"booked_at"`
}

type LedgerAccountAggregate struct {
	event.AggregateRoot

	AccountID          string                          `json:"account_id"`
	Balances           map[string]int64                `json:"balances"`
	PostedTransactions map[string]LedgerAccountPosting `json:"posted_transactions"`
}

func NewLedgerAccountAggregate(accountID string) (*LedgerAccountAggregate, error) {
	agg := &LedgerAccountAggregate{}
	if err := event.InitAggregate(&agg.AggregateRoot, agg, accountID); err != nil {
		return nil, stackErr.Error(err)
	}
	agg.ensureState()
	return agg, nil
}

func (a *LedgerAccountAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(
		&EventLedgerAccountPaymentBooked{},
		&EventLedgerAccountTransferredToAccount{},
		&EventLedgerAccountReceivedTransfer{},
	)
}

func (a *LedgerAccountAggregate) Transition(evt event.Event) error {
	switch data := evt.EventData.(type) {
	case *EventLedgerAccountPaymentBooked:
		return a.applyPaymentBooked(evt.AggregateID, data)
	case *EventLedgerAccountTransferredToAccount:
		return a.applyTransferredToAccount(evt.AggregateID, data)
	case *EventLedgerAccountReceivedTransfer:
		return a.applyReceivedTransfer(evt.AggregateID, data)
	default:
		return event.ErrUnsupportedEventType
	}
}

func (a *LedgerAccountAggregate) Balance(currency string) int64 {
	if a == nil {
		return 0
	}
	a.ensureState()
	return a.Balances[strings.ToUpper(strings.TrimSpace(currency))]
}

func (a *LedgerAccountAggregate) BookPayment(
	transactionID string,
	paymentID string,
	counterpartyAccountID string,
	currency string,
	amountDelta int64,
	bookedAt time.Time,
) (bool, error) {
	return a.bookPaymentPosting(
		transactionID,
		entity.PaymentReferenceSucceeded,
		paymentID,
		counterpartyAccountID,
		currency,
		amountDelta,
		bookedAt,
	)
}

func (a *LedgerAccountAggregate) ReversePayment(
	transactionID string,
	referenceType string,
	paymentID string,
	counterpartyAccountID string,
	currency string,
	amountDelta int64,
	bookedAt time.Time,
) (bool, error) {
	referenceType = strings.TrimSpace(referenceType)
	if referenceType != entity.PaymentReferenceRefunded && referenceType != entity.PaymentReferenceChargeback {
		return false, stackErr.Error(fmt.Errorf("payment reversal type is invalid: %s", referenceType))
	}

	return a.bookPaymentPosting(
		transactionID,
		referenceType,
		paymentID,
		counterpartyAccountID,
		currency,
		amountDelta,
		bookedAt,
	)
}

func (a *LedgerAccountAggregate) bookPaymentPosting(
	transactionID string,
	referenceType string,
	referenceID string,
	counterpartyAccountID string,
	currency string,
	amountDelta int64,
	bookedAt time.Time,
) (bool, error) {
	if a == nil {
		return false, stackErr.Error(ErrLedgerAccountAggregateRequired)
	}
	posting, err := newLedgerPosting(
		transactionID,
		referenceType,
		referenceID,
		counterpartyAccountID,
		currency,
		amountDelta,
		bookedAt,
	)
	if err != nil {
		return false, stackErr.Error(err)
	}

	existing, exists := a.lookupPosting(posting.TransactionID)
	if exists {
		if sameLedgerPosting(existing, posting) {
			return false, nil
		}
		return false, stackErr.Error(fmt.Errorf("ledger payment booking mismatch for transaction_id=%s", posting.TransactionID))
	}
	if err := a.ensurePostingAllowed(posting); err != nil {
		return false, stackErr.Error(err)
	}

	if err := a.ApplyChange(a, &EventLedgerAccountPaymentBooked{
		TransactionID:         posting.TransactionID,
		ReferenceType:         posting.ReferenceType,
		PaymentID:             posting.ReferenceID,
		CounterpartyAccountID: posting.CounterpartyAccountID,
		Currency:              posting.Currency,
		AmountDelta:           posting.AmountDelta,
		BookedAt:              posting.BookedAt,
	}); err != nil {
		return false, stackErr.Error(err)
	}

	return true, nil
}

func (a *LedgerAccountAggregate) TransferToAccount(
	transactionID string,
	toAccountID string,
	currency string,
	amount int64,
	bookedAt time.Time,
) (bool, error) {
	if a == nil {
		return false, stackErr.Error(ErrLedgerAccountAggregateRequired)
	}
	posting, err := newLedgerPosting(
		transactionID,
		"ledger.transfer_to_account",
		transactionID,
		toAccountID,
		currency,
		-amount,
		bookedAt,
	)
	if err != nil {
		return false, stackErr.Error(err)
	}

	existing, exists := a.lookupPosting(posting.TransactionID)
	if exists {
		if sameLedgerPosting(existing, posting) {
			return false, nil
		}
		return false, stackErr.Error(fmt.Errorf("ledger transfer mismatch for transaction_id=%s", posting.TransactionID))
	}
	if err := a.ensurePostingAllowed(posting); err != nil {
		return false, stackErr.Error(err)
	}

	if err := a.ApplyChange(a, &EventLedgerAccountTransferredToAccount{
		TransactionID: posting.TransactionID,
		ToAccountID:   posting.CounterpartyAccountID,
		Currency:      posting.Currency,
		Amount:        amount,
		BookedAt:      posting.BookedAt,
	}); err != nil {
		return false, stackErr.Error(err)
	}

	return true, nil
}

func (a *LedgerAccountAggregate) ReceiveTransfer(
	transactionID string,
	fromAccountID string,
	currency string,
	amount int64,
	bookedAt time.Time,
) (bool, error) {
	if a == nil {
		return false, stackErr.Error(ErrLedgerAccountAggregateRequired)
	}
	posting, err := newLedgerPosting(
		transactionID,
		"ledger.transfer_to_account",
		transactionID,
		fromAccountID,
		currency,
		amount,
		bookedAt,
	)
	if err != nil {
		return false, stackErr.Error(err)
	}

	existing, exists := a.lookupPosting(posting.TransactionID)
	if exists {
		if sameLedgerPosting(existing, posting) {
			return false, nil
		}
		return false, stackErr.Error(fmt.Errorf("ledger transfer receive mismatch for transaction_id=%s", posting.TransactionID))
	}

	if err := a.ApplyChange(a, &EventLedgerAccountReceivedTransfer{
		TransactionID: posting.TransactionID,
		FromAccountID: posting.CounterpartyAccountID,
		Currency:      posting.Currency,
		Amount:        amount,
		BookedAt:      posting.BookedAt,
	}); err != nil {
		return false, stackErr.Error(err)
	}

	return true, nil
}

func (a *LedgerAccountAggregate) applyPaymentBooked(accountID string, data *EventLedgerAccountPaymentBooked) error {
	if data == nil {
		return stackErr.Error(errors.New("ledger payment booked event is nil"))
	}

	referenceType := strings.TrimSpace(data.ReferenceType)
	if referenceType == "" {
		referenceType = entity.PaymentReferenceSucceeded
	}

	return a.applyPosting(accountID, LedgerAccountPosting{
		TransactionID:         strings.TrimSpace(data.TransactionID),
		ReferenceType:         referenceType,
		ReferenceID:           strings.TrimSpace(data.PaymentID),
		CounterpartyAccountID: strings.TrimSpace(data.CounterpartyAccountID),
		Currency:              strings.ToUpper(strings.TrimSpace(data.Currency)),
		AmountDelta:           data.AmountDelta,
		BookedAt:              data.BookedAt.UTC(),
	})
}

func (a *LedgerAccountAggregate) applyTransferredToAccount(accountID string, data *EventLedgerAccountTransferredToAccount) error {
	if data == nil {
		return stackErr.Error(errors.New("ledger transfer to account event is nil"))
	}

	return a.applyPosting(accountID, LedgerAccountPosting{
		TransactionID:         strings.TrimSpace(data.TransactionID),
		ReferenceType:         "ledger.transfer_to_account",
		ReferenceID:           strings.TrimSpace(data.TransactionID),
		CounterpartyAccountID: strings.TrimSpace(data.ToAccountID),
		Currency:              strings.ToUpper(strings.TrimSpace(data.Currency)),
		AmountDelta:           -data.Amount,
		BookedAt:              data.BookedAt.UTC(),
	})
}

func (a *LedgerAccountAggregate) applyReceivedTransfer(accountID string, data *EventLedgerAccountReceivedTransfer) error {
	if data == nil {
		return stackErr.Error(errors.New("ledger received transfer event is nil"))
	}

	return a.applyPosting(accountID, LedgerAccountPosting{
		TransactionID:         strings.TrimSpace(data.TransactionID),
		ReferenceType:         "ledger.transfer_to_account",
		ReferenceID:           strings.TrimSpace(data.TransactionID),
		CounterpartyAccountID: strings.TrimSpace(data.FromAccountID),
		Currency:              strings.ToUpper(strings.TrimSpace(data.Currency)),
		AmountDelta:           data.Amount,
		BookedAt:              data.BookedAt.UTC(),
	})
}

func (a *LedgerAccountAggregate) applyPosting(accountID string, posting LedgerAccountPosting) error {
	a.ensureState()
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return ErrLedgerAccountIDRequired
	}
	if a.AccountID != "" && a.AccountID != accountID {
		return fmt.Errorf("%w: aggregate=%s event=%s", ErrLedgerAccountIDMismatch, a.AccountID, accountID)
	}
	a.AccountID = accountID
	a.Balances[posting.Currency] += posting.AmountDelta
	a.PostedTransactions[posting.TransactionID] = posting
	return nil
}

func (a *LedgerAccountAggregate) ensurePostingAllowed(posting LedgerAccountPosting) error {
	if posting.AmountDelta >= 0 {
		return nil
	}
	if !requiresNonNegativeBalance(posting.ReferenceType) {
		return nil
	}

	nextBalance := a.Balance(posting.Currency) + posting.AmountDelta
	if nextBalance < 0 {
		return stackErr.Error(fmt.Errorf(
			"%w: account_id=%s currency=%s balance=%d amount=%d",
			ErrLedgerAccountInsufficientFunds,
			a.AggregateID(),
			posting.Currency,
			a.Balance(posting.Currency),
			-posting.AmountDelta,
		))
	}

	return nil
}

func requiresNonNegativeBalance(referenceType string) bool {
	switch strings.TrimSpace(referenceType) {
	case entity.PaymentReferenceSucceeded, entity.PaymentReferenceRefunded, entity.PaymentReferenceChargeback:
		// Provider settlement and provider reversals remain append-only; we do not silently mutate
		// historical postings just because the counterparty balance has already moved elsewhere.
		return false
	default:
		return true
	}
}

func (a *LedgerAccountAggregate) ensureState() {
	if a.Balances == nil {
		a.Balances = make(map[string]int64)
	}
	if a.PostedTransactions == nil {
		a.PostedTransactions = make(map[string]LedgerAccountPosting)
	}
}

func (a *LedgerAccountAggregate) lookupPosting(transactionID string) (LedgerAccountPosting, bool) {
	if a == nil {
		return LedgerAccountPosting{}, false
	}
	a.ensureState()
	posting, ok := a.PostedTransactions[strings.TrimSpace(transactionID)]
	return posting, ok
}

func newLedgerPosting(
	transactionID string,
	referenceType string,
	referenceID string,
	counterpartyAccountID string,
	currency string,
	amountDelta int64,
	bookedAt time.Time,
) (LedgerAccountPosting, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return LedgerAccountPosting{}, ErrLedgerAccountTransactionRequired
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return LedgerAccountPosting{}, ErrLedgerAccountCurrencyRequired
	}
	if amountDelta == 0 {
		return LedgerAccountPosting{}, ErrLedgerAccountAmountInvalid
	}
	if bookedAt.IsZero() {
		return LedgerAccountPosting{}, ErrLedgerAccountBookedAtRequired
	}

	return LedgerAccountPosting{
		TransactionID:         transactionID,
		ReferenceType:         strings.TrimSpace(referenceType),
		ReferenceID:           strings.TrimSpace(referenceID),
		CounterpartyAccountID: strings.TrimSpace(counterpartyAccountID),
		Currency:              currency,
		AmountDelta:           amountDelta,
		BookedAt:              bookedAt.UTC(),
	}, nil
}

func sameLedgerPosting(left LedgerAccountPosting, right LedgerAccountPosting) bool {
	return left.TransactionID == right.TransactionID &&
		left.ReferenceType == right.ReferenceType &&
		left.ReferenceID == right.ReferenceID &&
		left.CounterpartyAccountID == right.CounterpartyAccountID &&
		left.Currency == right.Currency &&
		left.AmountDelta == right.AmountDelta
}
