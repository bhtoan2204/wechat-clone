package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	ledgerin "go-socket/core/modules/ledger/application/dto/in"
	ledgerout "go-socket/core/modules/ledger/application/dto/out"
	"go-socket/core/modules/ledger/domain/entity"
	ledgerrepos "go-socket/core/modules/ledger/domain/repos"
	sharedevents "go-socket/core/shared/contracts/events"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type LedgerService struct {
	baseRepo ledgerrepos.Repos
}

func NewLedgerService(baseRepo ledgerrepos.Repos) *LedgerService {
	return &LedgerService{baseRepo: baseRepo}
}

func (s *LedgerService) CreateTransaction(ctx context.Context, req *ledgerin.CreateTransactionRequest) (*ledgerout.TransactionResponse, error) {
	if err := wrapValidation(req.Validate()); err != nil {
		return nil, stackErr.Error(err)
	}

	var transaction *entity.LedgerTransaction
	if err := s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		var err error
		transaction, err = s.createTransaction(ctx, txRepos.LedgerRepository(), req.TransactionID, toLedgerEntryInputs(req.Entries))
		return stackErr.Error(err)
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	return toTransactionResponse(transaction), nil
}

func (s *LedgerService) RecordPaymentSucceeded(ctx context.Context, evt *sharedevents.PaymentSucceededEvent) error {
	log := logging.FromContext(ctx).Named("RecordPaymentSucceeded")
	booking, err := entity.NewPaymentSucceededBooking(evt)
	if err != nil {
		return stackErr.Error(fmt.Errorf("%w: %s", ErrValidation, err.Error()))
	}

	return s.baseRepo.WithTransaction(ctx, func(txRepos ledgerrepos.Repos) error {
		processed, err := txRepos.PaymentRepository().IsProcessed(ctx, "payment-service", booking.IdempotencyKey)
		if err != nil {
			return stackErr.Error(err)
		}
		if processed {
			return nil
		}

		alreadyBooked := false
		if _, err := s.createTransaction(ctx, txRepos.LedgerRepository(), booking.LedgerTransactionID(), booking.LedgerEntries()); err != nil {
			if errors.Is(err, ErrDuplicateTransaction) {
				alreadyBooked = true
			} else {
				return stackErr.Error(err)
			}
		}

		processedEvent, err := booking.ProcessedEvent("payment-service", time.Now().UTC())
		if err != nil {
			return stackErr.Error(fmt.Errorf("%w: %s", ErrValidation, err.Error()))
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
	})
}

func (s *LedgerService) createTransaction(ctx context.Context, repo ledgerrepos.LedgerRepository, transactionID string, entries []entity.LedgerEntryInput) (*entity.LedgerTransaction, error) {
	log := logging.FromContext(ctx).Named("createTransaction")
	transaction, err := entity.NewLedgerTransaction(transactionID, entries, time.Now().UTC())
	if err != nil {
		return nil, stackErr.Error(fmt.Errorf("%w: %s", ErrValidation, err.Error()))
	}

	if err := repo.CreateTransaction(ctx, transaction); err != nil {
		if errors.Is(err, ledgerrepos.ErrDuplicate) {
			return nil, fmt.Errorf("%w: %s", ErrDuplicateTransaction, transaction.TransactionID)
		}
		return nil, stackErr.Error(err)
	}

	if err := repo.InsertEntries(ctx, transaction.Entries); err != nil {
		return nil, stackErr.Error(err)
	}

	transaction, err = repo.GetTransaction(ctx, transaction.TransactionID)
	if errors.Is(err, ledgerrepos.ErrNotFound) {
		return nil, fmt.Errorf("%w: %s", ErrTransactionNotFound, transactionID)
	}
	if err != nil {
		return nil, stackErr.Error(err)
	}

	log.Infow("ledger transaction created",
		zap.String("transaction_id", transaction.TransactionID),
		zap.Int("entries_count", len(transaction.Entries)),
	)

	return transaction, nil
}

func toLedgerEntryInputs(entries []ledgerin.LedgerEntryRequest) []entity.LedgerEntryInput {
	out := make([]entity.LedgerEntryInput, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entity.LedgerEntryInput{
			AccountID: strings.TrimSpace(entry.AccountID),
			Amount:    entry.Amount,
		})
	}
	return out
}

func toTransactionResponse(transaction *entity.LedgerTransaction) *ledgerout.TransactionResponse {
	entries := make([]ledgerout.LedgerEntryResponse, 0, len(transaction.Entries))
	for _, entry := range transaction.Entries {
		entries = append(entries, ledgerout.LedgerEntryResponse{
			ID:            entry.ID,
			TransactionID: entry.TransactionID,
			AccountID:     entry.AccountID,
			Amount:        entry.Amount,
			CreatedAt:     entry.CreatedAt,
		})
	}

	return &ledgerout.TransactionResponse{
		TransactionID: transaction.TransactionID,
		CreatedAt:     transaction.CreatedAt,
		Entries:       entries,
	}
}

func wrapValidation(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s", ErrValidation, err.Error())
}
