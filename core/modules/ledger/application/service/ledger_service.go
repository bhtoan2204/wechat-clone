package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	ledgerprojection "go-socket/core/modules/ledger/application/projection"
	ledgeraggregate "go-socket/core/modules/ledger/domain/aggregate"
	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type TransferToAccountCommand struct {
	TransactionID string
	FromAccountID string
	ToAccountID   string
	Currency      string
	Amount        int64
}

type RecordPaymentSucceededCommand struct {
	PaymentID          string
	TransactionID      string
	ClearingAccountKey string
	CreditAccountID    string
	Currency           string
	Amount             int64
}

type LedgerService interface {
	TransferToAccount(ctx context.Context, command TransferToAccountCommand) (*entity.LedgerTransaction, error)
	RecordPaymentSucceeded(ctx context.Context, command RecordPaymentSucceededCommand) error
}

type ledgerService struct {
	baseRepo ledgerrepos.Repos
}

func NewLedgerService(baseRepo ledgerrepos.Repos) *ledgerService {
	return &ledgerService{baseRepo: baseRepo}
}

func (s *ledgerService) TransferToAccount(ctx context.Context, command TransferToAccountCommand) (*entity.LedgerTransaction, error) {
	booking, err := entity.NewTransferBooking(entity.TransferBookingInput{
		FromAccountID: command.FromAccountID,
		ToAccountID:   command.ToAccountID,
		Currency:      command.Currency,
		Amount:        command.Amount,
	})
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%v: %v", ErrValidation, err))
	}

	transaction, err := entity.NewLedgerTransaction(strings.TrimSpace(command.TransactionID), booking.LedgerEntries())
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%v: %v", ErrValidation, err))
	}

	if err := s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		fromAgg, err := s.loadLedgerAccount(ctx, txRepos, booking.FromAccountID)
		if err != nil {
			return stackErr.Error(err)
		}
		toAgg, err := s.loadLedgerAccount(ctx, txRepos, booking.ToAccountID)
		if err != nil {
			return stackErr.Error(err)
		}

		fromApplied, err := fromAgg.TransferToAccount(
			transaction.TransactionID,
			booking.ToAccountID,
			transaction.Currency,
			booking.Amount,
			transaction.CreatedAt,
		)
		if err != nil {
			if errors.Is(err, ledgeraggregate.ErrLedgerAccountInsufficientFunds) {
				return stackErr.Error(fmt.Errorf("%v: %v", ErrInsufficientFunds, err))
			}
			return stackErr.Error(err)
		}
		toApplied, err := toAgg.ReceiveTransfer(
			transaction.TransactionID,
			booking.FromAccountID,
			transaction.Currency,
			booking.Amount,
			transaction.CreatedAt,
		)
		if err != nil {
			return stackErr.Error(err)
		}
		if fromApplied != toApplied {
			return stackErr.Error(fmt.Errorf("ledger transfer posting became inconsistent for transaction_id=%s", transaction.TransactionID))
		}
		if !fromApplied {
			return nil
		}

		if err := txRepos.LedgerAccountAggregateRepository().Save(ctx, fromAgg); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.LedgerAccountAggregateRepository().Save(ctx, toAgg); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.LedgerOutboxEventsRepository().Append(ctx, newLedgerTransactionProjectedEvent(
			transaction,
			"ledger.transfer_to_account",
			transaction.TransactionID,
		)); err != nil {
			return stackErr.Error(err)
		}

		return nil
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return transaction, nil
}

func (s *ledgerService) RecordPaymentSucceeded(ctx context.Context, command RecordPaymentSucceededCommand) error {
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
		return stackErr.Error(fmt.Errorf("%v: %v", ErrValidation, err))
	}

	transaction, err := entity.NewLedgerTransaction(booking.LedgerTransactionID(), booking.LedgerEntries())
	if err != nil {
		return stackErr.Error(fmt.Errorf("%v: %v", ErrValidation, err))
	}

	alreadyBooked := false
	if err := s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		debitAgg, err := s.loadLedgerAccount(ctx, txRepos, booking.DebitAccountID)
		if err != nil {
			return stackErr.Error(err)
		}
		creditAgg, err := s.loadLedgerAccount(ctx, txRepos, booking.CreditAccountID)
		if err != nil {
			return stackErr.Error(err)
		}

		debitApplied, err := debitAgg.BookPayment(
			transaction.TransactionID,
			booking.PaymentID,
			booking.CreditAccountID,
			transaction.Currency,
			-booking.Amount,
			transaction.CreatedAt,
		)
		if err != nil {
			return stackErr.Error(err)
		}
		creditApplied, err := creditAgg.BookPayment(
			transaction.TransactionID,
			booking.PaymentID,
			booking.DebitAccountID,
			transaction.Currency,
			booking.Amount,
			transaction.CreatedAt,
		)
		if err != nil {
			return stackErr.Error(err)
		}
		if debitApplied != creditApplied {
			return stackErr.Error(fmt.Errorf("ledger payment booking became inconsistent for transaction_id=%s", transaction.TransactionID))
		}
		if !debitApplied {
			alreadyBooked = true
			return nil
		}

		if err := txRepos.LedgerAccountAggregateRepository().Save(ctx, debitAgg); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.LedgerAccountAggregateRepository().Save(ctx, creditAgg); err != nil {
			return stackErr.Error(err)
		}
		if err := txRepos.LedgerOutboxEventsRepository().Append(ctx, newLedgerTransactionProjectedEvent(
			transaction,
			"payment.succeeded",
			booking.PaymentID,
		)); err != nil {
			return stackErr.Error(err)
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

func (s *ledgerService) loadLedgerAccount(
	ctx context.Context,
	repos ledgerrepos.Repos,
	accountID string,
) (*ledgeraggregate.LedgerAccountAggregate, error) {
	account, err := repos.LedgerAccountAggregateRepository().Load(ctx, accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if account != nil {
		return account, nil
	}
	account, err = ledgeraggregate.NewLedgerAccountAggregate(accountID)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return account, nil
}

func newLedgerTransactionProjectedEvent(
	transaction *entity.LedgerTransaction,
	referenceType string,
	referenceID string,
) eventpkg.Event {
	entries := make([]ledgerprojection.LedgerTransactionEntry, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		if entry == nil {
			continue
		}
		entries = append(entries, ledgerprojection.LedgerTransactionEntry{
			AccountID: entry.AccountID,
			Currency:  entry.Currency,
			Amount:    entry.Amount,
			CreatedAt: entry.CreatedAt.UTC(),
		})
	}

	return eventpkg.Event{
		AggregateID:   transaction.TransactionID,
		AggregateType: ledgerprojection.AggregateTypeLedgerTransactionProjection,
		Version:       1,
		EventName:     ledgerprojection.EventLedgerTransactionProjected,
		EventData: &ledgerprojection.LedgerTransactionProjected{
			TransactionID: transaction.TransactionID,
			ReferenceType: strings.TrimSpace(referenceType),
			ReferenceID:   strings.TrimSpace(referenceID),
			Currency:      transaction.Currency,
			CreatedAt:     transaction.CreatedAt.UTC(),
			Entries:       entries,
		},
		CreatedAt: transaction.CreatedAt.Unix(),
	}
}
