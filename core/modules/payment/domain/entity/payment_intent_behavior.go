package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sharedevents "go-socket/core/shared/contracts/events"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

var (
	ErrPaymentProviderRequired          = errors.New("provider is required")
	ErrPaymentTransactionIDRequired     = errors.New("transaction_id is required")
	ErrPaymentAmountInvalid             = errors.New("amount must be greater than 0")
	ErrPaymentCurrencyRequired          = errors.New("currency is required")
	ErrPaymentDebitAccountRequired      = errors.New("debit_account_id is required")
	ErrPaymentCreditAccountRequired     = errors.New("credit_account_id is required")
	ErrPaymentAccountsMustDiffer        = errors.New("debit_account_id and credit_account_id must be different")
	ErrPaymentStatusInvalid             = errors.New("status is invalid")
	ErrPaymentProviderAmountMismatch    = errors.New("provider amount does not match reserved payment")
	ErrPaymentProviderCurrencyMismatch  = errors.New("provider currency does not match reserved payment")
	ErrPaymentProcessedProviderRequired = errors.New("provider is required")
	ErrPaymentProcessedKeyRequired      = errors.New("idempotency_key is required")
	ErrPaymentProcessedTxnRequired      = errors.New("transaction_id is required")
)

func NewPaymentIntent(transactionID, provider string, amount int64, currency, debitAccountID, creditAccountID string, now time.Time) (*PaymentIntent, error) {
	transactionID = strings.TrimSpace(transactionID)
	provider = strings.ToLower(strings.TrimSpace(provider))
	currency = strings.ToUpper(strings.TrimSpace(currency))
	debitAccountID = strings.TrimSpace(debitAccountID)
	creditAccountID = strings.TrimSpace(creditAccountID)

	switch {
	case provider == "":
		return nil, ErrPaymentProviderRequired
	case transactionID == "":
		return nil, ErrPaymentTransactionIDRequired
	case amount <= 0:
		return nil, ErrPaymentAmountInvalid
	case currency == "":
		return nil, ErrPaymentCurrencyRequired
	case debitAccountID == "":
		return nil, ErrPaymentDebitAccountRequired
	case creditAccountID == "":
		return nil, ErrPaymentCreditAccountRequired
	case debitAccountID == creditAccountID:
		return nil, ErrPaymentAccountsMustDiffer
	}

	now = normalizePaymentTime(now)
	return &PaymentIntent{
		TransactionID:   transactionID,
		Provider:        provider,
		Amount:          amount,
		Currency:        currency,
		DebitAccountID:  debitAccountID,
		CreditAccountID: creditAccountID,
		Status:          PaymentStatusCreating,
		CreatedAt:       now,
		UpdatedAt:       now,
	}, nil
}

func NormalizePaymentStatus(status string) string {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case PaymentStatusCreating:
		return PaymentStatusCreating
	case PaymentStatusPending:
		return PaymentStatusPending
	case PaymentStatusSuccess:
		return PaymentStatusSuccess
	case PaymentStatusFailed:
		return PaymentStatusFailed
	default:
		return ""
	}
}

func NormalizePaymentStatusOrPending(status string) string {
	if normalized := NormalizePaymentStatus(status); normalized != "" {
		return normalized
	}
	return PaymentStatusPending
}

func (p *PaymentIntent) SetProviderState(externalRef, status string, updatedAt time.Time) error {
	if p == nil {
		return ErrPaymentTransactionIDRequired
	}

	normalizedStatus := NormalizePaymentStatus(status)
	if normalizedStatus == "" {
		return ErrPaymentStatusInvalid
	}

	if externalRef = strings.TrimSpace(externalRef); externalRef != "" {
		p.ExternalRef = externalRef
	}
	p.Status = normalizedStatus
	p.UpdatedAt = normalizePaymentTime(updatedAt)
	return nil
}

func (p *PaymentIntent) ApplyProviderResult(result PaymentProviderResult, updatedAt time.Time) error {
	if err := p.ValidateProviderResult(result.Amount, result.Currency); err != nil {
		return stackErr.Error(err)
	}

	return p.SetProviderState(result.ExternalRef, NormalizePaymentStatusOrPending(result.Status), updatedAt)
}

func (p *PaymentIntent) CurrentProviderResult(source PaymentProviderResult) PaymentProviderResult {
	if p == nil {
		return PaymentProviderResult{}
	}

	amount := source.Amount
	if amount == 0 {
		amount = p.Amount
	}

	return PaymentProviderResult{
		TransactionID: coalescePaymentValue(source.TransactionID, p.TransactionID),
		EventID:       strings.TrimSpace(source.EventID),
		EventType:     strings.TrimSpace(source.EventType),
		Status:        NormalizePaymentStatusOrPending(coalescePaymentValue(source.Status, p.Status)),
		Amount:        amount,
		Currency:      coalescePaymentValue(source.Currency, p.Currency),
		ExternalRef:   coalescePaymentValue(source.ExternalRef, p.ExternalRef),
	}
}

func (p *PaymentIntent) MarkCreateFailed(updatedAt time.Time) error {
	return p.SetProviderState("", PaymentStatusFailed, updatedAt)
}

func (p *PaymentIntent) IsSucceeded() bool {
	return p != nil && p.Status == PaymentStatusSuccess
}

func (p *PaymentIntent) IsFailed() bool {
	return p != nil && p.Status == PaymentStatusFailed
}

func (p *PaymentIntent) ShouldEmitCheckoutSessionCreated(checkoutURL string) bool {
	if p == nil {
		return false
	}
	return strings.TrimSpace(checkoutURL) != "" || strings.TrimSpace(p.ExternalRef) != ""
}

func (p *PaymentIntent) ValidateProviderResult(amount int64, currency string) error {
	if p == nil {
		return ErrPaymentTransactionIDRequired
	}
	if amount != 0 && amount != p.Amount {
		return ErrPaymentProviderAmountMismatch
	}
	if currency = strings.TrimSpace(currency); currency != "" && !strings.EqualFold(currency, p.Currency) {
		return ErrPaymentProviderCurrencyMismatch
	}
	return nil
}

func (p *PaymentIntent) PaymentIdempotencyKey(eventID, externalRef string) string {
	if eventID = strings.TrimSpace(eventID); eventID != "" {
		return eventID
	}
	if externalRef = strings.TrimSpace(externalRef); externalRef != "" {
		return externalRef
	}
	if externalRef = strings.TrimSpace(p.ExternalRef); externalRef != "" {
		return externalRef
	}
	return strings.TrimSpace(p.TransactionID)
}

func (p *PaymentIntent) CreatedEvent(metadata map[string]string, createdAt time.Time) eventpkg.Event {
	occurredAt := normalizePaymentTime(createdAt)
	return eventpkg.Event{
		AggregateID:   p.TransactionID,
		AggregateType: PaymentAggregateType,
		Version:       1,
		EventName:     sharedevents.EventPaymentCreated,
		EventData: sharedevents.PaymentCreatedEvent{
			PaymentID:       p.TransactionID,
			TransactionID:   p.TransactionID,
			Provider:        p.Provider,
			Amount:          p.Amount,
			Currency:        p.Currency,
			DebitAccountID:  p.DebitAccountID,
			CreditAccountID: p.CreditAccountID,
			Status:          p.Status,
			Metadata:        metadata,
			CreatedAt:       occurredAt,
		},
		CreatedAt: occurredAt.Unix(),
	}
}

func (p *PaymentIntent) CheckoutSessionCreatedEvent(checkoutURL string, occurredAt time.Time) eventpkg.Event {
	eventTime := normalizePaymentTime(occurredAt)
	return eventpkg.Event{
		AggregateID:   p.TransactionID,
		AggregateType: PaymentAggregateType,
		Version:       1,
		EventName:     sharedevents.EventPaymentCheckoutSessionCreated,
		EventData: sharedevents.PaymentCheckoutSessionCreatedEvent{
			PaymentID:          p.TransactionID,
			TransactionID:      p.TransactionID,
			Provider:           p.Provider,
			ProviderPaymentRef: p.ExternalRef,
			CheckoutURL:        strings.TrimSpace(checkoutURL),
			Amount:             p.Amount,
			Currency:           p.Currency,
			Status:             p.Status,
			OccurredAt:         eventTime,
		},
		CreatedAt: eventTime.Unix(),
	}
}

func (p *PaymentIntent) SucceededEvent(result PaymentProviderResult, occurredAt time.Time) eventpkg.Event {
	eventTime := normalizePaymentTime(occurredAt)
	return eventpkg.Event{
		AggregateID:   p.TransactionID,
		AggregateType: PaymentAggregateType,
		Version:       1,
		EventName:     sharedevents.EventPaymentSucceeded,
		EventData: sharedevents.PaymentSucceededEvent{
			PaymentID:          p.TransactionID,
			TransactionID:      p.TransactionID,
			Provider:           p.Provider,
			ProviderEventID:    strings.TrimSpace(result.EventID),
			ProviderEventType:  strings.TrimSpace(result.EventType),
			ProviderPaymentRef: coalescePaymentValue(result.ExternalRef, p.ExternalRef),
			Amount:             p.Amount,
			Currency:           p.Currency,
			DebitAccountID:     p.DebitAccountID,
			CreditAccountID:    p.CreditAccountID,
			IdempotencyKey:     fmt.Sprintf("%s:%s", sharedevents.EventPaymentSucceeded, p.TransactionID),
			SucceededAt:        eventTime,
		},
		CreatedAt: eventTime.Unix(),
	}
}

func (p *PaymentIntent) FailedEvent(result PaymentProviderResult, occurredAt time.Time) eventpkg.Event {
	eventTime := normalizePaymentTime(occurredAt)
	return eventpkg.Event{
		AggregateID:   p.TransactionID,
		AggregateType: PaymentAggregateType,
		Version:       1,
		EventName:     sharedevents.EventPaymentFailed,
		EventData: sharedevents.PaymentFailedEvent{
			PaymentID:          p.TransactionID,
			TransactionID:      p.TransactionID,
			Provider:           p.Provider,
			ProviderEventID:    strings.TrimSpace(result.EventID),
			ProviderEventType:  strings.TrimSpace(result.EventType),
			ProviderPaymentRef: coalescePaymentValue(result.ExternalRef, p.ExternalRef),
			Amount:             p.Amount,
			Currency:           p.Currency,
			Status:             NormalizePaymentStatusOrPending(result.Status),
			OccurredAt:         eventTime,
		},
		CreatedAt: eventTime.Unix(),
	}
}

func (p *PaymentIntent) NewProcessedEvent(result PaymentProviderResult, createdAt time.Time) (*ProcessedPaymentEvent, error) {
	return NewProcessedPaymentEvent(
		p.Provider,
		p.PaymentIdempotencyKey(result.EventID, result.ExternalRef),
		p.TransactionID,
		createdAt,
	)
}

func NewProcessedPaymentEvent(provider, idempotencyKey, transactionID string, createdAt time.Time) (*ProcessedPaymentEvent, error) {
	provider = strings.TrimSpace(provider)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	transactionID = strings.TrimSpace(transactionID)

	switch {
	case provider == "":
		return nil, ErrPaymentProcessedProviderRequired
	case idempotencyKey == "":
		return nil, ErrPaymentProcessedKeyRequired
	case transactionID == "":
		return nil, ErrPaymentProcessedTxnRequired
	}

	return &ProcessedPaymentEvent{
		Provider:       provider,
		IdempotencyKey: idempotencyKey,
		TransactionID:  transactionID,
		CreatedAt:      normalizePaymentTime(createdAt),
	}, nil
}

func normalizePaymentTime(value time.Time) time.Time {
	if value.IsZero() {
		return time.Now().UTC()
	}
	return value.UTC()
}

func coalescePaymentValue(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}
