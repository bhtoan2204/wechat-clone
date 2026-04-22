package aggregate

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"wechat-clone/core/modules/ledger/domain/entity"
	valueobject "wechat-clone/core/modules/ledger/domain/value_object"
	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/event"
	"wechat-clone/core/shared/pkg/stackErr"
)

var (
	ErrLedgerAccountAggregateRequired    = errors.New("ledger account aggregate is required")
	ErrLedgerAccountIDRequired           = errors.New("ledger account id is required")
	ErrLedgerAccountIDMismatch           = errors.New("ledger account id mismatch")
	ErrLedgerAccountTransactionRequired  = errors.New("ledger transaction id is required")
	ErrLedgerAccountReferenceTypeInvalid = errors.New("ledger reference type is invalid")
	ErrLedgerAccountReferenceIDRequired  = errors.New("ledger reference id is required")
	ErrLedgerAccountCounterpartyRequired = errors.New("ledger counterparty_account_id is required")
	ErrLedgerAccountAccountsMustDiffer   = errors.New("ledger account_id and counterparty_account_id must be different")
	ErrLedgerAccountCurrencyRequired     = errors.New("ledger currency is required")
	ErrLedgerAccountAmountInvalid        = errors.New("ledger amount must be positive")
	ErrLedgerAccountBookedAtRequired     = errors.New("ledger booked_at is required")
	ErrLedgerAccountInsufficientFunds    = errors.New("ledger account has insufficient funds")
)

type LedgerAccountAggregate struct {
	event.AggregateRoot

	AccountID          string                                 `json:"account_id"`
	Balances           map[string]int64                       `json:"balances"`
	PostedTransactions map[string]entity.LedgerAccountPosting `json:"posted_transactions"`
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
		&EventLedgerAccountDepositFromIntent{},
		&EventLedgerAccountWithdrawFromIntent{},
		&EventLedgerAccountDepositFromRefund{},
		&EventLedgerAccountWithdrawFromRefund{},
		&EventLedgerAccountDepositFromChargeback{},
		&EventLedgerAccountWithdrawFromChargeback{},
		&EventLedgerAccountTransferredToAccount{},
		&EventLedgerAccountReceivedTransfer{},
	)
}

func (a *LedgerAccountAggregate) Transition(evt event.Event) error {
	switch data := evt.EventData.(type) {
	case *EventLedgerAccountDepositFromIntent:
		return a.applyDepositFromIntent(evt.AggregateID, data)
	case *EventLedgerAccountWithdrawFromIntent:
		return a.applyWithdrawFromIntent(evt.AggregateID, data)
	case *EventLedgerAccountDepositFromRefund:
		return a.applyDepositFromRefund(evt.AggregateID, data)
	case *EventLedgerAccountWithdrawFromRefund:
		return a.applyWithdrawFromRefund(evt.AggregateID, data)
	case *EventLedgerAccountDepositFromChargeback:
		return a.applyDepositFromChargeback(evt.AggregateID, data)
	case *EventLedgerAccountWithdrawFromChargeback:
		return a.applyWithdrawFromChargeback(evt.AggregateID, data)
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

func (a *LedgerAccountAggregate) PostedTransaction(transactionID string) (*entity.LedgerAccountPosting, bool) {
	if a == nil {
		return nil, false
	}
	a.ensureState()
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return nil, false
	}

	posting, ok := a.PostedTransactions[transactionID]
	if !ok {
		return nil, false
	}
	copyPosting := posting
	return &copyPosting, true
}

func (a *LedgerAccountAggregate) ApplyPostingEvent(eventData interface{}) (bool, error) {
	if a == nil {
		return false, stackErr.Error(ErrLedgerAccountAggregateRequired)
	}

	posting, ok, err := previewLedgerPostingFromEvent(a.AggregateID(), eventData)
	if err != nil {
		return false, stackErr.Error(err)
	}
	if !ok {
		return false, stackErr.Error(event.ErrUnsupportedEventType)
	}

	existing, exists := a.lookupPendingPosting(posting.TransactionID)
	if exists {
		if SameLedgerAccountPosting(existing, posting) {
			return false, nil
		}
		return false, stackErr.Error(fmt.Errorf("ledger posting mismatch for transaction_id=%s", posting.TransactionID))
	}
	if err := a.ensurePostingAllowed(posting); err != nil {
		return false, stackErr.Error(err)
	}

	if err := a.ApplyChange(a, eventData); err != nil {
		return false, stackErr.Error(err)
	}
	return true, nil
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
		ledgerPaymentReferenceTypeForSucceededAmount(amountDelta),
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
	if referenceType != sharedevents.EventPaymentRefunded && referenceType != sharedevents.EventPaymentChargeback {
		return false, stackErr.Error(fmt.Errorf("%w: %s", ErrLedgerAccountReferenceTypeInvalid, referenceType))
	}

	return a.bookPaymentPosting(
		transactionID,
		ledgerPaymentReferenceTypeForReversal(referenceType, amountDelta),
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
	posting, err := NewLedgerAccountPaymentPosting(
		valueobject.LedgerAccountPostingInput{
			AccountID:             a.AggregateID(),
			TransactionID:         transactionID,
			ReferenceType:         referenceType,
			ReferenceID:           referenceID,
			CounterpartyAccountID: counterpartyAccountID,
			Currency:              currency,
			AmountDelta:           amountDelta,
			BookedAt:              bookedAt,
		},
	)
	if err != nil {
		return false, stackErr.Error(err)
	}

	existing, exists := a.lookupPendingPosting(posting.TransactionID)
	if exists {
		if SameLedgerAccountPosting(existing, posting) {
			return false, nil
		}
		return false, stackErr.Error(fmt.Errorf("ledger payment booking mismatch for transaction_id=%s", posting.TransactionID))
	}
	if err := a.ensurePostingAllowed(posting); err != nil {
		return false, stackErr.Error(err)
	}

	if err := a.ApplyChange(a, newLedgerPaymentEvent(posting)); err != nil {
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
	posting, err := NewLedgerAccountTransferOutPosting(
		valueobject.LedgerAccountTransferPostingInput{
			AccountID:             a.AggregateID(),
			TransactionID:         transactionID,
			CounterpartyAccountID: toAccountID,
			Currency:              currency,
			Amount:                amount,
			BookedAt:              bookedAt,
		},
	)
	if err != nil {
		return false, stackErr.Error(err)
	}

	existing, exists := a.lookupPendingPosting(posting.TransactionID)
	if exists {
		if SameLedgerAccountPosting(existing, posting) {
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
	posting, err := NewLedgerAccountTransferInPosting(
		valueobject.LedgerAccountTransferPostingInput{
			AccountID:             a.AggregateID(),
			TransactionID:         transactionID,
			CounterpartyAccountID: fromAccountID,
			Currency:              currency,
			Amount:                amount,
			BookedAt:              bookedAt,
		},
	)
	if err != nil {
		return false, stackErr.Error(err)
	}

	existing, exists := a.lookupPendingPosting(posting.TransactionID)
	if exists {
		if SameLedgerAccountPosting(existing, posting) {
			return false, nil
		}
		return false, stackErr.Error(fmt.Errorf("ledger transfer receive mismatch for transaction_id=%s", posting.TransactionID))
	}
	if err := a.ensurePostingAllowed(posting); err != nil {
		return false, stackErr.Error(err)
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

func (a *LedgerAccountAggregate) applyTransferredToAccount(accountID string, data *EventLedgerAccountTransferredToAccount) error {
	if data == nil {
		return stackErr.Error(errors.New("ledger transfer to account event is nil"))
	}
	posting, err := NewLedgerAccountTransferOutPosting(valueobject.LedgerAccountTransferPostingInput{
		AccountID:             accountID,
		TransactionID:         data.TransactionID,
		CounterpartyAccountID: data.ToAccountID,
		Currency:              data.Currency,
		Amount:                data.Amount,
		BookedAt:              data.BookedAt,
	})
	if err != nil {
		return stackErr.Error(err)
	}
	return a.applyPosting(accountID, posting)
}

func (a *LedgerAccountAggregate) applyDepositFromIntent(accountID string, data *EventLedgerAccountDepositFromIntent) error {
	return a.applyEventPosting(accountID, data, "ledger deposit from intent event is unsupported")
}

func (a *LedgerAccountAggregate) applyWithdrawFromIntent(accountID string, data *EventLedgerAccountWithdrawFromIntent) error {
	return a.applyEventPosting(accountID, data, "ledger withdraw from intent event is unsupported")
}

func (a *LedgerAccountAggregate) applyDepositFromRefund(accountID string, data *EventLedgerAccountDepositFromRefund) error {
	return a.applyEventPosting(accountID, data, "ledger deposit from refund event is unsupported")
}

func (a *LedgerAccountAggregate) applyWithdrawFromRefund(accountID string, data *EventLedgerAccountWithdrawFromRefund) error {
	return a.applyEventPosting(accountID, data, "ledger withdraw from refund event is unsupported")
}

func (a *LedgerAccountAggregate) applyDepositFromChargeback(accountID string, data *EventLedgerAccountDepositFromChargeback) error {
	return a.applyEventPosting(accountID, data, "ledger deposit from chargeback event is unsupported")
}

func (a *LedgerAccountAggregate) applyWithdrawFromChargeback(accountID string, data *EventLedgerAccountWithdrawFromChargeback) error {
	return a.applyEventPosting(accountID, data, "ledger withdraw from chargeback event is unsupported")
}

func (a *LedgerAccountAggregate) applyEventPosting(accountID string, eventData interface{}, unsupportedErr string) error {
	posting, ok, err := previewLedgerPostingFromEvent(accountID, eventData)
	if err != nil {
		return stackErr.Error(err)
	}
	if !ok {
		return stackErr.Error(errors.New(unsupportedErr))
	}
	return a.applyPosting(accountID, posting)
}

func (a *LedgerAccountAggregate) applyReceivedTransfer(accountID string, data *EventLedgerAccountReceivedTransfer) error {
	if data == nil {
		return stackErr.Error(errors.New("ledger received transfer event is nil"))
	}
	posting, err := NewLedgerAccountTransferInPosting(valueobject.LedgerAccountTransferPostingInput{
		AccountID:             accountID,
		TransactionID:         data.TransactionID,
		CounterpartyAccountID: data.FromAccountID,
		Currency:              data.Currency,
		Amount:                data.Amount,
		BookedAt:              data.BookedAt,
	})
	if err != nil {
		return stackErr.Error(err)
	}
	return a.applyPosting(accountID, posting)
}

func (a *LedgerAccountAggregate) applyPosting(accountID string, posting entity.LedgerAccountPosting) error {
	normalizedAccountID, normalizedPosting, err := normalizeLedgerAccountPosting(accountID, posting)
	if err != nil {
		return err
	}
	if err := a.ensurePostingAllowed(normalizedPosting); err != nil {
		return err
	}

	a.ensureState()
	if a.AccountID != "" && a.AccountID != normalizedAccountID {
		return fmt.Errorf("%w: aggregate=%s event=%s", ErrLedgerAccountIDMismatch, a.AccountID, normalizedAccountID)
	}

	a.AccountID = normalizedAccountID
	a.Balances[normalizedPosting.Currency] += normalizedPosting.AmountDelta
	a.PostedTransactions[normalizedPosting.TransactionID] = normalizedPosting
	return nil
}

func (a *LedgerAccountAggregate) ensurePostingAllowed(posting entity.LedgerAccountPosting) error {
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
	case EventNameLedgerAccountDepositFromIntent,
		EventNameLedgerAccountWithdrawFromIntent,
		EventNameLedgerAccountDepositFromRefund,
		EventNameLedgerAccountWithdrawFromRefund,
		EventNameLedgerAccountDepositFromChargeback,
		EventNameLedgerAccountWithdrawFromChargeback:
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
		a.PostedTransactions = make(map[string]entity.LedgerAccountPosting)
	}
}

func (a *LedgerAccountAggregate) lookupPendingPosting(transactionID string) (entity.LedgerAccountPosting, bool) {
	if a == nil {
		return entity.LedgerAccountPosting{}, false
	}
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return entity.LedgerAccountPosting{}, false
	}
	if existing, ok := a.PostedTransactions[transactionID]; ok {
		return existing, true
	}

	return entity.LedgerAccountPosting{}, false
}

func normalizeLedgerAccountPosting(accountID string, posting entity.LedgerAccountPosting) (string, entity.LedgerAccountPosting, error) {
	accountID = strings.TrimSpace(accountID)
	if accountID == "" {
		return "", entity.LedgerAccountPosting{}, ErrLedgerAccountIDRequired
	}

	normalizedReferenceType := strings.TrimSpace(posting.ReferenceType)
	switch normalizedReferenceType {
	case EventNameLedgerAccountDepositFromIntent,
		EventNameLedgerAccountWithdrawFromIntent,
		EventNameLedgerAccountDepositFromRefund,
		EventNameLedgerAccountWithdrawFromRefund,
		EventNameLedgerAccountDepositFromChargeback,
		EventNameLedgerAccountWithdrawFromChargeback,
		entity.LedgerReferenceInternalTransfer:
	default:
		return "", entity.LedgerAccountPosting{}, ErrLedgerAccountReferenceTypeInvalid
	}

	normalizedTransactionID := strings.TrimSpace(posting.TransactionID)
	if normalizedTransactionID == "" {
		return "", entity.LedgerAccountPosting{}, ErrLedgerAccountTransactionRequired
	}

	normalizedReferenceID := strings.TrimSpace(posting.ReferenceID)
	if normalizedReferenceID == "" {
		return "", entity.LedgerAccountPosting{}, ErrLedgerAccountReferenceIDRequired
	}

	normalizedCounterpartyAccountID := strings.TrimSpace(posting.CounterpartyAccountID)
	if normalizedCounterpartyAccountID == "" {
		return "", entity.LedgerAccountPosting{}, ErrLedgerAccountCounterpartyRequired
	}
	if normalizedCounterpartyAccountID == accountID {
		return "", entity.LedgerAccountPosting{}, ErrLedgerAccountAccountsMustDiffer
	}

	normalizedCurrency := strings.ToUpper(strings.TrimSpace(posting.Currency))
	if normalizedCurrency == "" {
		return "", entity.LedgerAccountPosting{}, ErrLedgerAccountCurrencyRequired
	}
	if posting.AmountDelta == 0 {
		return "", entity.LedgerAccountPosting{}, ErrLedgerAccountAmountInvalid
	}
	if posting.BookedAt.IsZero() {
		return "", entity.LedgerAccountPosting{}, ErrLedgerAccountBookedAtRequired
	}

	return accountID, entity.LedgerAccountPosting{
		TransactionID:         normalizedTransactionID,
		ReferenceType:         normalizedReferenceType,
		ReferenceID:           normalizedReferenceID,
		CounterpartyAccountID: normalizedCounterpartyAccountID,
		Currency:              normalizedCurrency,
		AmountDelta:           posting.AmountDelta,
		BookedAt:              posting.BookedAt.UTC(),
	}, nil
}

func NewLedgerAccountPaymentPosting(input valueobject.LedgerAccountPostingInput) (entity.LedgerAccountPosting, error) {
	posting, err := newLedgerPosting(
		input.TransactionID,
		input.ReferenceType,
		input.ReferenceID,
		input.CounterpartyAccountID,
		input.Currency,
		input.AmountDelta,
		input.BookedAt,
	)
	if err != nil {
		return entity.LedgerAccountPosting{}, stackErr.Error(err)
	}

	_, normalizedPosting, err := normalizeLedgerAccountPosting(input.AccountID, posting)
	if err != nil {
		return entity.LedgerAccountPosting{}, stackErr.Error(err)
	}

	return normalizedPosting, nil
}

func NewLedgerAccountTransferOutPosting(input valueobject.LedgerAccountTransferPostingInput) (entity.LedgerAccountPosting, error) {
	return NewLedgerAccountPaymentPosting(
		valueobject.LedgerAccountPostingInput{
			AccountID:             input.AccountID,
			TransactionID:         input.TransactionID,
			ReferenceType:         entity.LedgerReferenceInternalTransfer,
			ReferenceID:           input.TransactionID,
			CounterpartyAccountID: input.CounterpartyAccountID,
			Currency:              input.Currency,
			AmountDelta:           -input.Amount,
			BookedAt:              input.BookedAt,
		},
	)
}

func NewLedgerAccountTransferInPosting(input valueobject.LedgerAccountTransferPostingInput) (entity.LedgerAccountPosting, error) {
	return NewLedgerAccountPaymentPosting(
		valueobject.LedgerAccountPostingInput{
			AccountID:             input.AccountID,
			TransactionID:         input.TransactionID,
			ReferenceType:         entity.LedgerReferenceInternalTransfer,
			ReferenceID:           input.TransactionID,
			CounterpartyAccountID: input.CounterpartyAccountID,
			Currency:              input.Currency,
			AmountDelta:           input.Amount,
			BookedAt:              input.BookedAt,
		},
	)
}

func NewLedgerAccountPostingFromEvent(accountID string, eventData interface{}) (entity.LedgerAccountPosting, bool, error) {
	return previewLedgerPostingFromEvent(accountID, eventData)
}

func previewLedgerPostingFromEvent(accountID string, eventData interface{}) (entity.LedgerAccountPosting, bool, error) {
	switch data := eventData.(type) {
	case *EventLedgerAccountDepositFromIntent:
		return newLedgerPostingFromPaymentEvent(accountID, data, EventNameLedgerAccountDepositFromIntent, 1, "ledger deposit from intent event is nil")
	case *EventLedgerAccountWithdrawFromIntent:
		return newLedgerPostingFromPaymentEvent(accountID, data, EventNameLedgerAccountWithdrawFromIntent, -1, "ledger withdraw from intent event is nil")
	case *EventLedgerAccountDepositFromRefund:
		return newLedgerPostingFromPaymentEvent(accountID, data, EventNameLedgerAccountDepositFromRefund, 1, "ledger deposit from refund event is nil")
	case *EventLedgerAccountWithdrawFromRefund:
		return newLedgerPostingFromPaymentEvent(accountID, data, EventNameLedgerAccountWithdrawFromRefund, -1, "ledger withdraw from refund event is nil")
	case *EventLedgerAccountDepositFromChargeback:
		return newLedgerPostingFromPaymentEvent(accountID, data, EventNameLedgerAccountDepositFromChargeback, 1, "ledger deposit from chargeback event is nil")
	case *EventLedgerAccountWithdrawFromChargeback:
		return newLedgerPostingFromPaymentEvent(accountID, data, EventNameLedgerAccountWithdrawFromChargeback, -1, "ledger withdraw from chargeback event is nil")
	case *EventLedgerAccountTransferredToAccount:
		if data == nil {
			return entity.LedgerAccountPosting{}, false, stackErr.Error(errors.New("ledger transfer to account event is nil"))
		}

		posting, err := NewLedgerAccountTransferOutPosting(valueobject.LedgerAccountTransferPostingInput{
			AccountID:             accountID,
			TransactionID:         data.TransactionID,
			CounterpartyAccountID: data.ToAccountID,
			Currency:              data.Currency,
			Amount:                data.Amount,
			BookedAt:              data.BookedAt,
		})
		return posting, true, stackErr.Error(err)
	case *EventLedgerAccountReceivedTransfer:
		if data == nil {
			return entity.LedgerAccountPosting{}, false, stackErr.Error(errors.New("ledger received transfer event is nil"))
		}

		posting, err := NewLedgerAccountTransferInPosting(valueobject.LedgerAccountTransferPostingInput{
			AccountID:             accountID,
			TransactionID:         data.TransactionID,
			CounterpartyAccountID: data.FromAccountID,
			Currency:              data.Currency,
			Amount:                data.Amount,
			BookedAt:              data.BookedAt,
		})
		return posting, true, stackErr.Error(err)
	default:
		return entity.LedgerAccountPosting{}, false, nil
	}
}

type ledgerPaymentEventData struct {
	transactionID         string
	paymentID             string
	counterpartyAccountID string
	currency              string
	amount                int64
	bookedAt              time.Time
}

func newLedgerPaymentEvent(posting entity.LedgerAccountPosting) interface{} {
	base := ledgerPaymentEventData{
		transactionID:         posting.TransactionID,
		paymentID:             posting.ReferenceID,
		counterpartyAccountID: posting.CounterpartyAccountID,
		currency:              posting.Currency,
		amount:                absInt64(posting.AmountDelta),
		bookedAt:              posting.BookedAt,
	}

	switch posting.ReferenceType {
	case EventNameLedgerAccountDepositFromIntent:
		return &EventLedgerAccountDepositFromIntent{
			TransactionID:         base.transactionID,
			PaymentID:             base.paymentID,
			CounterpartyAccountID: base.counterpartyAccountID,
			Currency:              base.currency,
			Amount:                base.amount,
			BookedAt:              base.bookedAt,
		}
	case EventNameLedgerAccountWithdrawFromIntent:
		return &EventLedgerAccountWithdrawFromIntent{
			TransactionID:         base.transactionID,
			PaymentID:             base.paymentID,
			CounterpartyAccountID: base.counterpartyAccountID,
			Currency:              base.currency,
			Amount:                base.amount,
			BookedAt:              base.bookedAt,
		}
	case EventNameLedgerAccountDepositFromRefund:
		return &EventLedgerAccountDepositFromRefund{
			TransactionID:         base.transactionID,
			PaymentID:             base.paymentID,
			CounterpartyAccountID: base.counterpartyAccountID,
			Currency:              base.currency,
			Amount:                base.amount,
			BookedAt:              base.bookedAt,
		}
	case EventNameLedgerAccountWithdrawFromRefund:
		return &EventLedgerAccountWithdrawFromRefund{
			TransactionID:         base.transactionID,
			PaymentID:             base.paymentID,
			CounterpartyAccountID: base.counterpartyAccountID,
			Currency:              base.currency,
			Amount:                base.amount,
			BookedAt:              base.bookedAt,
		}
	case EventNameLedgerAccountDepositFromChargeback:
		return &EventLedgerAccountDepositFromChargeback{
			TransactionID:         base.transactionID,
			PaymentID:             base.paymentID,
			CounterpartyAccountID: base.counterpartyAccountID,
			Currency:              base.currency,
			Amount:                base.amount,
			BookedAt:              base.bookedAt,
		}
	case EventNameLedgerAccountWithdrawFromChargeback:
		return &EventLedgerAccountWithdrawFromChargeback{
			TransactionID:         base.transactionID,
			PaymentID:             base.paymentID,
			CounterpartyAccountID: base.counterpartyAccountID,
			Currency:              base.currency,
			Amount:                base.amount,
			BookedAt:              base.bookedAt,
		}
	default:
		return nil
	}
}

func newLedgerPostingFromPaymentEvent(
	accountID string,
	data interface{},
	referenceType string,
	direction int64,
	nilErr string,
) (entity.LedgerAccountPosting, bool, error) {
	if data == nil {
		return entity.LedgerAccountPosting{}, false, stackErr.Error(errors.New(nilErr))
	}

	eventData, ok := data.(interface {
		paymentEvent() ledgerPaymentEventData
	})
	if !ok {
		return entity.LedgerAccountPosting{}, false, nil
	}

	payload := eventData.paymentEvent()
	posting, err := NewLedgerAccountPaymentPosting(valueobject.LedgerAccountPostingInput{
		AccountID:             accountID,
		TransactionID:         payload.transactionID,
		ReferenceType:         referenceType,
		ReferenceID:           payload.paymentID,
		CounterpartyAccountID: payload.counterpartyAccountID,
		Currency:              payload.currency,
		AmountDelta:           direction * payload.amount,
		BookedAt:              payload.bookedAt,
	})
	return posting, true, stackErr.Error(err)
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func newLedgerPosting(
	transactionID string,
	referenceType string,
	referenceID string,
	counterpartyAccountID string,
	currency string,
	amountDelta int64,
	bookedAt time.Time,
) (entity.LedgerAccountPosting, error) {
	transactionID = strings.TrimSpace(transactionID)
	if transactionID == "" {
		return entity.LedgerAccountPosting{}, ErrLedgerAccountTransactionRequired
	}
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if currency == "" {
		return entity.LedgerAccountPosting{}, ErrLedgerAccountCurrencyRequired
	}
	if amountDelta == 0 {
		return entity.LedgerAccountPosting{}, ErrLedgerAccountAmountInvalid
	}
	if bookedAt.IsZero() {
		return entity.LedgerAccountPosting{}, ErrLedgerAccountBookedAtRequired
	}

	return entity.LedgerAccountPosting{
		TransactionID:         transactionID,
		ReferenceType:         strings.TrimSpace(referenceType),
		ReferenceID:           strings.TrimSpace(referenceID),
		CounterpartyAccountID: strings.TrimSpace(counterpartyAccountID),
		Currency:              currency,
		AmountDelta:           amountDelta,
		BookedAt:              bookedAt.UTC(),
	}, nil
}

func ledgerPaymentReferenceTypeForSucceededAmount(amountDelta int64) string {
	if amountDelta >= 0 {
		return EventNameLedgerAccountDepositFromIntent
	}
	return EventNameLedgerAccountWithdrawFromIntent
}

func ledgerPaymentReferenceTypeForReversal(paymentEventName string, amountDelta int64) string {
	switch strings.TrimSpace(paymentEventName) {
	case sharedevents.EventPaymentRefunded:
		if amountDelta >= 0 {
			return EventNameLedgerAccountDepositFromRefund
		}
		return EventNameLedgerAccountWithdrawFromRefund
	case sharedevents.EventPaymentChargeback:
		if amountDelta >= 0 {
			return EventNameLedgerAccountDepositFromChargeback
		}
		return EventNameLedgerAccountWithdrawFromChargeback
	default:
		return ""
	}
}

func SameLedgerAccountPosting(left entity.LedgerAccountPosting, right entity.LedgerAccountPosting) bool {
	return left.TransactionID == right.TransactionID &&
		left.ReferenceType == right.ReferenceType &&
		left.ReferenceID == right.ReferenceID &&
		left.CounterpartyAccountID == right.CounterpartyAccountID &&
		left.Currency == right.Currency &&
		left.AmountDelta == right.AmountDelta
}
