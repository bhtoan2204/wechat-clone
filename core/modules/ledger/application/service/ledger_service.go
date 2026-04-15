package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ledgeraggregate "go-socket/core/modules/ledger/domain/aggregate"
	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type CreateTransactionEntryCommand struct {
	AccountID string
	Amount    int64
}

type CreateTransactionCommand struct {
	TransactionID string
	Currency      string
	Entries       []CreateTransactionEntryCommand
}

type RecordPaymentSucceededCommand struct {
	PaymentID          string
	TransactionID      string
	ClearingAccountKey string
	CreditAccountID    string
	Currency           string
	Amount             int64
}

type LedgerService struct {
	baseRepo ledgerrepos.Repos
}

func NewLedgerService(baseRepo ledgerrepos.Repos) *LedgerService {
	return &LedgerService{baseRepo: baseRepo}
}

func (s *LedgerService) CreateTransaction(ctx context.Context, command CreateTransactionCommand) (*entity.LedgerTransaction, error) {
	log := logging.FromContext(ctx).Named("CreateLedgerTransaction")
	transactionID := strings.TrimSpace(command.TransactionID)

	aggregate, err := ledgeraggregate.NewLedgerTransactionAggregate(transactionID)
	if err != nil {
		return nil, stackErr.Error(wrapValidation(err))
	}
	if err := aggregate.Create(toLedgerEntryInputs(command.Entries, command.Currency), time.Now().UTC()); err != nil {
		return nil, stackErr.Error(wrapValidation(err))
	}

	if err := s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		if err := txRepos.LedgerTransactionAggregateRepository().Save(ctx, aggregate); err != nil {
			if errors.Is(err, ledgerrepos.ErrDuplicate) {
				return stackErr.Error(fmt.Errorf("%w: %s", ErrDuplicateTransaction, transactionID))
			}
			return stackErr.Error(err)
		}
		return nil
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	transaction, err := aggregate.Snapshot()
	if err != nil {
		return nil, stackErr.Error(err)
	}

	log.Infow("ledger transaction created",
		zap.String("transaction_id", transaction.TransactionID),
		zap.Int("entries_count", len(transaction.Entries)),
	)

	return transaction, nil
}

func (s *LedgerService) RecordPaymentSucceeded(ctx context.Context, command RecordPaymentSucceededCommand) error {
	log := logging.FromContext(ctx).Named("RecordPaymentSucceeded")
	booking, err := entity.NewPaymentSucceededBooking(entity.PaymentSucceededBookingInput{
		PaymentID:          command.PaymentID,
		TransactionID:      command.TransactionID,
		ClearingAccountKey: command.ClearingAccountKey,
		CreditAccountID:    command.CreditAccountID,
		Currency:           command.Currency,
		Amount:             command.Amount,
	})
	if err != nil {
		return stackErr.Error(wrapValidation(err))
	}

	now := time.Now().UTC()
	alreadyBooked := false
	if err := s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		transactionAggregate, err := ledgeraggregate.NewLedgerTransactionAggregate(booking.LedgerTransactionID())
		if err != nil {
			return stackErr.Error(wrapValidation(err))
		}
		if err := transactionAggregate.Create(booking.LedgerEntries(), now); err != nil {
			return stackErr.Error(wrapValidation(err))
		}
		// Duplicate payment events must reconcile against the canonical ledger
		// transaction instead of persisting provider-oriented dedupe state here.
		if err := txRepos.LedgerTransactionAggregateRepository().Save(ctx, transactionAggregate); err != nil {
			if errors.Is(err, ledgerrepos.ErrDuplicate) {
				alreadyBooked = true
				return stackErr.Error(s.ensureExistingPaymentBooking(ctx, txRepos, booking))
			} else {
				return stackErr.Error(err)
			}
		}

		return nil
	}); err != nil {
		return stackErr.Error(err)
	}

	log.Infow("payment booked into ledger",
		zap.String("payment_id", booking.PaymentID),
		zap.String("ledger_transaction_id", booking.LedgerTransactionID()),
		zap.Bool("already_booked", alreadyBooked),
	)

	return nil
}

func toLedgerEntryInputs(entries []CreateTransactionEntryCommand, currency string) []entity.LedgerEntryInput {
	out := make([]entity.LedgerEntryInput, 0, len(entries))
	currency = strings.ToUpper(strings.TrimSpace(currency))
	for _, entry := range entries {
		out = append(out, entity.LedgerEntryInput{
			AccountID: strings.TrimSpace(entry.AccountID),
			Currency:  currency,
			Amount:    entry.Amount,
		})
	}
	return out
}

func wrapValidation(err error) error {
	if err == nil {
		return nil
	}
	return stackErr.Error(fmt.Errorf("%w: %s", ErrValidation, err.Error()))
}

func (s *LedgerService) ensureExistingPaymentBooking(
	ctx context.Context,
	repos ledgerrepos.Repos,
	booking *entity.PaymentSucceededBooking,
) error {
	transaction, err := repos.LedgerRepository().GetTransaction(ctx, booking.LedgerTransactionID())
	if err != nil {
		return stackErr.Error(err)
	}
	if bookingMatchesTransaction(booking, transaction) {
		return nil
	}

	return stackErr.Error(fmt.Errorf(
		"existing ledger transaction does not match payment booking: %s",
		booking.LedgerTransactionID(),
	))
}

func bookingMatchesTransaction(
	booking *entity.PaymentSucceededBooking,
	transaction *entity.LedgerTransaction,
) bool {
	if booking == nil || transaction == nil {
		return false
	}
	if strings.TrimSpace(transaction.TransactionID) != booking.LedgerTransactionID() {
		return false
	}
	if strings.ToUpper(strings.TrimSpace(transaction.Currency)) != booking.Currency {
		return false
	}

	expectedEntries := booking.LedgerEntries()
	if len(transaction.Entries) != len(expectedEntries) {
		return false
	}

	for i, expectedEntry := range expectedEntries {
		actualEntry := transaction.Entries[i]
		if actualEntry == nil {
			return false
		}
		if strings.ToUpper(strings.TrimSpace(actualEntry.Currency)) != expectedEntry.Currency {
			return false
		}
		if strings.TrimSpace(actualEntry.AccountID) != expectedEntry.AccountID {
			return false
		}
		if actualEntry.Amount != expectedEntry.Amount {
			return false
		}
	}

	return true
}
