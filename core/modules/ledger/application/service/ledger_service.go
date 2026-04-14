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

const paymentSucceededSource = "payment-service"

type CreateTransactionEntryCommand struct {
	AccountID string
	Amount    int64
}

type CreateTransactionCommand struct {
	TransactionID string
	Entries       []CreateTransactionEntryCommand
}

type RecordPaymentSucceededCommand struct {
	PaymentID       string
	TransactionID   string
	DebitAccountID  string
	CreditAccountID string
	Amount          int64
	IdempotencyKey  string
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
	if err := aggregate.Create(toLedgerEntryInputs(command.Entries), time.Now().UTC()); err != nil {
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
		PaymentID:       command.PaymentID,
		TransactionID:   command.TransactionID,
		DebitAccountID:  command.DebitAccountID,
		CreditAccountID: command.CreditAccountID,
		Amount:          command.Amount,
		IdempotencyKey:  command.IdempotencyKey,
	})
	if err != nil {
		return stackErr.Error(wrapValidation(err))
	}

	now := time.Now().UTC()
	return stackErr.Error(s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		processed, err := txRepos.PaymentRepository().IsProcessed(ctx, paymentSucceededSource, booking.IdempotencyKey)
		if err != nil {
			return stackErr.Error(err)
		}
		if processed {
			return nil
		}

		alreadyBooked := false
		transactionAggregate, err := ledgeraggregate.NewLedgerTransactionAggregate(booking.LedgerTransactionID())
		if err != nil {
			return stackErr.Error(wrapValidation(err))
		}
		if err := transactionAggregate.Create(booking.LedgerEntries(), now); err != nil {
			return stackErr.Error(wrapValidation(err))
		}
		if err := txRepos.LedgerTransactionAggregateRepository().Save(ctx, transactionAggregate); err != nil {
			if errors.Is(err, ledgerrepos.ErrDuplicate) {
				alreadyBooked = true
			} else {
				return stackErr.Error(err)
			}
		}

		processedEvent, err := booking.ProcessedEvent(paymentSucceededSource, now)
		if err != nil {
			return stackErr.Error(wrapValidation(err))
		}
		if err := txRepos.PaymentRepository().MarkProcessed(ctx, processedEvent); err != nil {
			if errors.Is(err, ledgerrepos.ErrDuplicate) {
				return nil
			}
			return stackErr.Error(err)
		}

		log.Infow("payment booked into ledger",
			zap.String("payment_id", booking.PaymentID),
			zap.String("ledger_transaction_id", booking.LedgerTransactionID()),
			zap.String("idempotency_key", booking.IdempotencyKey),
			zap.Bool("already_booked", alreadyBooked),
		)
		return nil
	}))
}

func toLedgerEntryInputs(entries []CreateTransactionEntryCommand) []entity.LedgerEntryInput {
	out := make([]entity.LedgerEntryInput, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entity.LedgerEntryInput{
			AccountID: strings.TrimSpace(entry.AccountID),
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
