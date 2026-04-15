package aggregate

import (
	"errors"
	"reflect"
	"strings"
	"time"

	"go-socket/core/modules/ledger/domain/entity"
	"go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

var (
	ErrLedgerTransactionAggregateRequired = errors.New("ledger transaction aggregate is required")
	ErrLedgerTransactionAlreadyCreated    = errors.New("ledger transaction already created")
)

type LedgerTransactionAggregate struct {
	event.AggregateRoot

	TransactionID string
	Currency      string
	CreatedAt     time.Time
	Entries       []*entity.LedgerEntry
}

func NewLedgerTransactionAggregate(transactionID string) (*LedgerTransactionAggregate, error) {
	agg := &LedgerTransactionAggregate{}
	agg.SetAggregateType(reflect.TypeOf(agg).Elem().Name())
	if err := agg.SetID(strings.TrimSpace(transactionID)); err != nil {
		return nil, stackErr.Error(err)
	}

	return agg, nil
}

func (a *LedgerTransactionAggregate) RegisterEvents(register event.RegisterEventsFunc) error {
	return register(&EventLedgerTransactionCreated{})
}

func (a *LedgerTransactionAggregate) Transition(evt event.Event) error {
	switch data := evt.EventData.(type) {
	case *EventLedgerTransactionCreated:
		return a.onLedgerTransactionCreated(evt.AggregateID, data)
	default:
		return stackErr.Error(errors.New("unsupported event type"))
	}
}

func (a *LedgerTransactionAggregate) Create(entries []entity.LedgerEntryInput, now time.Time) error {
	if a == nil {
		return stackErr.Error(ErrLedgerTransactionAggregateRequired)
	}
	if !a.CreatedAt.IsZero() {
		return stackErr.Error(ErrLedgerTransactionAlreadyCreated)
	}

	transaction, err := entity.NewLedgerTransaction(a.AggregateID(), entries, now)
	if err != nil {
		return stackErr.Error(err)
	}

	payloadEntries := make([]LedgerTransactionEntryPayload, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		if entry == nil {
			continue
		}

		payloadEntries = append(payloadEntries, LedgerTransactionEntryPayload{
			AccountID: entry.AccountID,
			Amount:    entry.Amount,
		})
	}

	return a.ApplyChange(a, &EventLedgerTransactionCreated{
		TransactionID: transaction.TransactionID,
		Currency:      transaction.Currency,
		CreatedAt:     transaction.CreatedAt,
		Entries:       payloadEntries,
	})
}

func (a *LedgerTransactionAggregate) Snapshot() (*entity.LedgerTransaction, error) {
	if a == nil || a.TransactionID == "" {
		return nil, stackErr.Error(ErrLedgerTransactionAggregateRequired)
	}

	entries := make([]*entity.LedgerEntry, 0, len(a.Entries))
	for _, entry := range a.Entries {
		if entry == nil {
			continue
		}

		entryCopy := *entry
		entries = append(entries, &entryCopy)
	}

	return &entity.LedgerTransaction{
		TransactionID: a.TransactionID,
		Currency:      a.Currency,
		CreatedAt:     a.CreatedAt,
		Entries:       entries,
	}, nil
}

func (a *LedgerTransactionAggregate) AssignEntryIDs(entryIDs []int64) error {
	if a == nil {
		return stackErr.Error(ErrLedgerTransactionAggregateRequired)
	}
	if len(entryIDs) != len(a.Entries) {
		return stackErr.Error(errors.New("entry ids count mismatch"))
	}

	for idx, entryID := range entryIDs {
		if a.Entries[idx] == nil {
			return stackErr.Error(errors.New("ledger entry is nil"))
		}
		a.Entries[idx].ID = entryID
	}

	return nil
}

func (a *LedgerTransactionAggregate) onLedgerTransactionCreated(
	transactionID string,
	data *EventLedgerTransactionCreated,
) error {
	if data == nil {
		return stackErr.Error(errors.New("ledger transaction created event is nil"))
	}

	a.TransactionID = transactionID
	a.Currency = entityNormalizeLedgerCurrency(data.Currency)
	a.CreatedAt = data.CreatedAt
	a.Entries = make([]*entity.LedgerEntry, 0, len(data.Entries))
	for _, entry := range data.Entries {
		a.Entries = append(a.Entries, &entity.LedgerEntry{
			TransactionID: transactionID,
			AccountID:     entry.AccountID,
			Currency:      a.Currency,
			Amount:        entry.Amount,
			CreatedAt:     data.CreatedAt,
		})
	}

	return nil
}

func entityNormalizeLedgerCurrency(value string) string {
	return strings.ToUpper(strings.TrimSpace(value))
}
