package entity

import (
	"errors"
	"fmt"
	"strings"
	"time"

	sharedevents "wechat-clone/core/shared/contracts/events"
	"wechat-clone/core/shared/pkg/stackErr"
)

var (
	ErrPaymentProviderRequired           = errors.New("provider is required")
	ErrPaymentTransactionIDRequired      = errors.New("transaction_id is required")
	ErrPaymentWorkflowInvalid            = errors.New("workflow is invalid")
	ErrPaymentAmountInvalid              = errors.New("amount must be greater than 0")
	ErrPaymentFeeAmountInvalid           = errors.New("fee_amount must be greater than or equal to 0")
	ErrPaymentProviderAmountInvalid      = errors.New("provider_amount must be greater than 0")
	ErrPaymentCurrencyRequired           = errors.New("currency is required")
	ErrPaymentClearingAccountKeyMissing  = errors.New("clearing_account_key is required")
	ErrPaymentDestinationAccountRequired = errors.New("destination_account_id is required")
	ErrPaymentDebitAccountRequired       = errors.New("debit_account_id is required")
	ErrPaymentCreditAccountRequired      = errors.New("credit_account_id is required")
	ErrPaymentAccountsConflict           = errors.New("debit_account_id and credit_account_id must be different")
	ErrPaymentStatusInvalid              = errors.New("status is invalid")
	ErrPaymentProviderAmountMismatch     = errors.New("provider amount does not match reserved payment")
	ErrPaymentProviderCurrencyMismatch   = errors.New("provider currency does not match reserved payment")
	ErrPaymentProcessedProviderRequired  = errors.New("provider is required")
	ErrPaymentProcessedKeyRequired       = errors.New("idempotency_key is required")
	ErrPaymentProcessedTxnRequired       = errors.New("transaction_id is required")
)

const (
	PaymentWorkflowTopUp      = "TOP_UP"
	PaymentWorkflowWithdrawal = "WITHDRAWAL"
)

func NewProviderTopUpIntent(
	transactionID,
	provider string,
	amount int64,
	feeAmount int64,
	currency,
	beneficiaryAccountID string,
	now time.Time,
) (*PaymentIntent, error) {
	return newPaymentIntent(
		PaymentWorkflowTopUp,
		transactionID,
		provider,
		amount,
		feeAmount,
		amount+feeAmount,
		currency,
		providerClearingAccountKey(provider),
		"",
		"",
		beneficiaryAccountID,
		now,
	)
}

func NewProviderWithdrawalIntent(
	transactionID,
	provider string,
	amount int64,
	feeAmount int64,
	currency,
	destinationAccountID,
	debitAccountID string,
	now time.Time,
) (*PaymentIntent, error) {
	return newPaymentIntent(
		PaymentWorkflowWithdrawal,
		transactionID,
		provider,
		amount,
		feeAmount,
		amount,
		currency,
		providerClearingAccountKey(provider),
		destinationAccountID,
		debitAccountID,
		"",
		now,
	)
}

func newPaymentIntent(workflow, transactionID, provider string, amount int64, feeAmount int64, providerAmount int64, currency, clearingAccountKey, destinationAccountID, debitAccountID, creditAccountID string, now time.Time) (*PaymentIntent, error) {
	workflow = NormalizePaymentWorkflow(workflow)
	transactionID = strings.TrimSpace(transactionID)
	provider = strings.ToLower(strings.TrimSpace(provider))
	currency = strings.ToUpper(strings.TrimSpace(currency))
	clearingAccountKey = effectivePaymentClearingAccountKey(provider, clearingAccountKey)
	destinationAccountID = strings.TrimSpace(destinationAccountID)
	debitAccountID = strings.TrimSpace(debitAccountID)
	creditAccountID = strings.TrimSpace(creditAccountID)

	switch {
	case workflow == "":
		return nil, ErrPaymentWorkflowInvalid
	case provider == "":
		return nil, ErrPaymentProviderRequired
	case transactionID == "":
		return nil, ErrPaymentTransactionIDRequired
	case amount <= 0:
		return nil, ErrPaymentAmountInvalid
	case feeAmount < 0:
		return nil, ErrPaymentFeeAmountInvalid
	case providerAmount <= 0:
		return nil, ErrPaymentProviderAmountInvalid
	case currency == "":
		return nil, ErrPaymentCurrencyRequired
	case clearingAccountKey == "":
		return nil, ErrPaymentClearingAccountKeyMissing
	case workflow == PaymentWorkflowTopUp && creditAccountID == "":
		return nil, ErrPaymentCreditAccountRequired
	case workflow == PaymentWorkflowWithdrawal && debitAccountID == "":
		return nil, ErrPaymentDebitAccountRequired
	case workflow == PaymentWorkflowWithdrawal && destinationAccountID == "":
		return nil, ErrPaymentDestinationAccountRequired
	case debitAccountID != "" && creditAccountID != "" && debitAccountID == creditAccountID:
		return nil, ErrPaymentAccountsConflict
	}

	now = normalizePaymentTime(now)
	return &PaymentIntent{
		Workflow:             workflow,
		TransactionID:        transactionID,
		Provider:             provider,
		DestinationAccountID: destinationAccountID,
		Amount:               amount,
		FeeAmount:            feeAmount,
		ProviderAmount:       providerAmount,
		Currency:             currency,
		ClearingAccountKey:   clearingAccountKey,
		DebitAccountID:       debitAccountID,
		CreditAccountID:      creditAccountID,
		Status:               PaymentStatusCreating,
		CreatedAt:            now,
		UpdatedAt:            now,
	}, nil
}

func providerClearingAccountKey(provider string) string {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if provider == "" {
		return ""
	}
	return fmt.Sprintf("provider:%s", provider)
}

func effectivePaymentClearingAccountKey(provider, clearingAccountKey string) string {
	if clearingAccountKey = strings.TrimSpace(clearingAccountKey); clearingAccountKey != "" {
		return clearingAccountKey
	}
	return providerClearingAccountKey(provider)
}

func NormalizePaymentWorkflow(workflow string) string {
	switch strings.ToUpper(strings.TrimSpace(workflow)) {
	case PaymentWorkflowTopUp:
		return PaymentWorkflowTopUp
	case PaymentWorkflowWithdrawal:
		return PaymentWorkflowWithdrawal
	default:
		return ""
	}
}

func NormalizePaymentStatus(status string) string {
	normalized := strings.ToUpper(strings.TrimSpace(status))
	return ValidPaymentStatuses[normalized]
}

func NormalizePaymentStatusOrPending(status string) string {
	if normalized := NormalizePaymentStatus(status); normalized != "" {
		return normalized
	}
	return PaymentStatusPending
}

func (p *PaymentIntent) SetProviderState(externalRef, status string, updatedAt time.Time) error {
	_, err := p.transitionProviderState(externalRef, status, updatedAt)
	return err
}

func (p *PaymentIntent) TransitionProviderResult(result PaymentProviderResult, updatedAt time.Time) (PaymentTransition, error) {
	if p != nil {
		p.ensureWorkflowDefaults()
	}
	nextStatus := NormalizePaymentStatusOrPending(result.Status)
	if err := p.ValidateProviderResultForStatus(nextStatus, result.Amount, result.Currency); err != nil {
		return PaymentTransition{}, stackErr.Error(err)
	}

	return p.transitionProviderState(result.ExternalRef, nextStatus, updatedAt)
}

func (p *PaymentIntent) ApplyProviderResult(result PaymentProviderResult, updatedAt time.Time) error {
	_, err := p.TransitionProviderResult(result, updatedAt)
	return err
}

func (p *PaymentIntent) transitionProviderState(externalRef, status string, updatedAt time.Time) (PaymentTransition, error) {
	if p == nil {
		return PaymentTransition{}, ErrPaymentTransactionIDRequired
	}
	p.ensureWorkflowDefaults()

	normalizedStatus := NormalizePaymentStatus(status)
	if normalizedStatus == "" {
		return PaymentTransition{}, ErrPaymentStatusInvalid
	}

	transition := resolvePaymentTransition(NormalizePaymentStatusOrPending(p.Status), normalizedStatus)
	if transition.StateChanged {
		p.Status = transition.CurrentStatus
	}

	if externalRef = strings.TrimSpace(externalRef); externalRef != "" && externalRef != p.ExternalRef {
		p.ExternalRef = externalRef
		transition.ExternalRefChanged = true
	}

	if transition.StateChanged || transition.ExternalRefChanged {
		p.UpdatedAt = normalizePaymentTime(updatedAt)
	}

	return transition, nil
}

func resolvePaymentTransition(previousStatus, nextStatus string) PaymentTransition {
	transition := PaymentTransition{
		PreviousStatus: NormalizePaymentStatusOrPending(previousStatus),
		CurrentStatus:  NormalizePaymentStatusOrPending(previousStatus),
		Type:           PaymentTransitionNone,
	}

	if transition.CurrentStatus == nextStatus {
		return transition
	}

	switch transition.CurrentStatus {
	case PaymentStatusCreating:
		switch nextStatus {
		case PaymentStatusPending:
			transition.CurrentStatus = nextStatus
			transition.Type = PaymentTransitionPending
			transition.StateChanged = true
		case PaymentStatusSuccess:
			transition.CurrentStatus = nextStatus
			transition.Type = PaymentTransitionSucceeded
			transition.StateChanged = true
		case PaymentStatusFailed:
			transition.CurrentStatus = nextStatus
			transition.Type = PaymentTransitionFailed
			transition.StateChanged = true
		case PaymentStatusCancelled:
			transition.CurrentStatus = nextStatus
			transition.Type = PaymentTransitionCancelled
			transition.StateChanged = true
		default:
			transition.Ignored = true
		}
	case PaymentStatusPending:
		switch nextStatus {
		case PaymentStatusSuccess:
			transition.CurrentStatus = nextStatus
			transition.Type = PaymentTransitionSucceeded
			transition.StateChanged = true
		case PaymentStatusFailed:
			transition.CurrentStatus = nextStatus
			transition.Type = PaymentTransitionFailed
			transition.StateChanged = true
		case PaymentStatusCancelled:
			transition.CurrentStatus = nextStatus
			transition.Type = PaymentTransitionCancelled
			transition.StateChanged = true
		default:
			transition.Ignored = true
		}
	case PaymentStatusSuccess:
		switch nextStatus {
		case PaymentStatusRefunded:
			transition.CurrentStatus = nextStatus
			transition.Type = PaymentTransitionRefunded
			transition.StateChanged = true
		case PaymentStatusChargeback:
			transition.CurrentStatus = nextStatus
			transition.Type = PaymentTransitionChargeback
			transition.StateChanged = true
		default:
			transition.Ignored = true
		}
	case PaymentStatusFailed, PaymentStatusCancelled, PaymentStatusRefunded, PaymentStatusChargeback:
		transition.Ignored = true
	default:
		transition.Ignored = true
	}

	return transition
}

func (p *PaymentIntent) MarkCreateFailed(updatedAt time.Time) (PaymentTransition, error) {
	return p.transitionProviderState("", PaymentStatusFailed, updatedAt)
}

func (p *PaymentIntent) IsSucceeded() bool {
	return p != nil && p.Status == PaymentStatusSuccess
}

func (p *PaymentIntent) IsTopUp() bool {
	return p != nil && p.Workflow == PaymentWorkflowTopUp
}

func (p *PaymentIntent) IsWithdrawal() bool {
	return p != nil && p.Workflow == PaymentWorkflowWithdrawal
}

func (p *PaymentIntent) BuildWithdrawalRequestedEventData(occurredAt time.Time) sharedevents.PaymentWithdrawalRequestedEvent {
	if p == nil {
		return sharedevents.PaymentWithdrawalRequestedEvent{}
	}

	occurredAt = normalizePaymentTime(occurredAt)
	return sharedevents.PaymentWithdrawalRequestedEvent{
		PaymentID:            p.TransactionID,
		TransactionID:        p.TransactionID,
		Provider:             p.Provider,
		ClearingAccountKey:   p.ClearingAccountKey,
		DebitAccountID:       p.DebitAccountID,
		DestinationAccountID: p.DestinationAccountID,
		Amount:               p.Amount,
		FeeAmount:            p.FeeAmount,
		ProviderAmount:       p.ProviderAmount,
		Currency:             p.Currency,
		Status:               p.Status,
		RequestedAt:          occurredAt,
	}
}

func (p *PaymentIntent) IsFailed() bool {
	return p != nil && p.Status == PaymentStatusFailed
}

func (p *PaymentIntent) IsCancelled() bool {
	return p != nil && p.Status == PaymentStatusCancelled
}

func (p *PaymentIntent) IsRefunded() bool {
	return p != nil && p.Status == PaymentStatusRefunded
}

func (p *PaymentIntent) IsChargeback() bool {
	return p != nil && p.Status == PaymentStatusChargeback
}

func (p *PaymentIntent) IsTerminal() bool {
	if p == nil {
		return false
	}

	switch p.Status {
	case PaymentStatusSuccess, PaymentStatusFailed, PaymentStatusCancelled, PaymentStatusRefunded, PaymentStatusChargeback:
		return true
	default:
		return false
	}
}

func (p *PaymentIntent) IsFinalized() bool {
	return p != nil && (p.IsSucceeded() || p.IsRefunded() || p.IsChargeback())
}

func (p *PaymentIntent) ShouldEmitCheckoutSessionCreated(checkoutURL string) bool {
	if p == nil {
		return false
	}
	if p.Workflow != PaymentWorkflowTopUp {
		return false
	}
	return strings.TrimSpace(checkoutURL) != "" || strings.TrimSpace(p.ExternalRef) != ""
}

func (p *PaymentIntent) ValidateProviderResult(amount int64, currency string) error {
	return p.ValidateProviderResultForStatus(p.Status, amount, currency)
}

func (p *PaymentIntent) ValidateProviderResultForStatus(status string, amount int64, currency string) error {
	if p == nil {
		return ErrPaymentTransactionIDRequired
	}

	normalizedStatus := NormalizePaymentStatus(status)
	if normalizedStatus == "" {
		normalizedStatus = NormalizePaymentStatusOrPending(status)
	}

	switch normalizedStatus {
	case PaymentStatusRefunded, PaymentStatusChargeback:
		if amount != 0 && amount > p.ProviderAmount {
			return ErrPaymentProviderAmountMismatch
		}
	default:
		if amount != 0 && amount != p.ProviderAmount {
			return ErrPaymentProviderAmountMismatch
		}
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

func (p *PaymentIntent) TransitionIdempotencyKey(eventName string) string {
	return fmt.Sprintf("%s:%s", strings.TrimSpace(eventName), strings.TrimSpace(p.TransactionID))
}

func (p *PaymentIntent) BuildCreatedEventData(metadata map[string]string, createdAt time.Time) sharedevents.PaymentCreatedEvent {
	p.ensureWorkflowDefaults()
	occurredAt := normalizePaymentTime(createdAt)
	return sharedevents.PaymentCreatedEvent{
		Workflow:           p.Workflow,
		PaymentID:          p.TransactionID,
		TransactionID:      p.TransactionID,
		Provider:           p.Provider,
		ClearingAccountKey: p.ClearingAccountKey,
		Amount:             p.Amount,
		FeeAmount:          p.FeeAmount,
		ProviderAmount:     p.ProviderAmount,
		Currency:           p.Currency,
		DebitAccountID:     p.DebitAccountID,
		CreditAccountID:    p.CreditAccountID,
		Status:             p.Status,
		Metadata:           metadata,
		CreatedAt:          occurredAt,
	}
}

func (p *PaymentIntent) BuildCheckoutSessionCreatedEventData(checkoutURL string, occurredAt time.Time) sharedevents.PaymentCheckoutSessionCreatedEvent {
	p.ensureWorkflowDefaults()
	eventTime := normalizePaymentTime(occurredAt)
	return sharedevents.PaymentCheckoutSessionCreatedEvent{
		Workflow:           p.Workflow,
		PaymentID:          p.TransactionID,
		TransactionID:      p.TransactionID,
		Provider:           p.Provider,
		ProviderPaymentRef: p.ExternalRef,
		CheckoutURL:        strings.TrimSpace(checkoutURL),
		Amount:             p.Amount,
		FeeAmount:          p.FeeAmount,
		ProviderAmount:     p.ProviderAmount,
		Currency:           p.Currency,
		Status:             p.Status,
		OccurredAt:         eventTime,
	}
}

func (p *PaymentIntent) BuildSucceededEventData(result PaymentProviderResult, occurredAt time.Time) sharedevents.PaymentSucceededEvent {
	p.ensureWorkflowDefaults()
	eventTime := normalizePaymentTime(occurredAt)
	return sharedevents.PaymentSucceededEvent{
		Workflow:           p.Workflow,
		PaymentID:          p.TransactionID,
		TransactionID:      p.TransactionID,
		Provider:           p.Provider,
		ClearingAccountKey: p.ClearingAccountKey,
		DebitAccountID:     p.DebitAccountID,
		ProviderEventID:    strings.TrimSpace(result.EventID),
		ProviderEventType:  strings.TrimSpace(result.EventType),
		ProviderPaymentRef: coalescePaymentValue(result.ExternalRef, p.ExternalRef),
		Amount:             p.Amount,
		FeeAmount:          p.FeeAmount,
		ProviderAmount:     p.ProviderAmount,
		Currency:           p.Currency,
		CreditAccountID:    p.CreditAccountID,
		IdempotencyKey:     p.TransitionIdempotencyKey(sharedevents.EventPaymentSucceeded),
		SucceededAt:        eventTime,
	}
}

func (p *PaymentIntent) BuildFailedEventData(result PaymentProviderResult, occurredAt time.Time) sharedevents.PaymentFailedEvent {
	p.ensureWorkflowDefaults()
	eventTime := normalizePaymentTime(occurredAt)
	return sharedevents.PaymentFailedEvent{
		Workflow:           p.Workflow,
		PaymentID:          p.TransactionID,
		TransactionID:      p.TransactionID,
		Provider:           p.Provider,
		ClearingAccountKey: p.ClearingAccountKey,
		DebitAccountID:     p.DebitAccountID,
		ProviderEventID:    strings.TrimSpace(result.EventID),
		ProviderEventType:  strings.TrimSpace(result.EventType),
		ProviderPaymentRef: coalescePaymentValue(result.ExternalRef, p.ExternalRef),
		Amount:             p.Amount,
		FeeAmount:          p.FeeAmount,
		ProviderAmount:     p.ProviderAmount,
		Currency:           p.Currency,
		CreditAccountID:    p.CreditAccountID,
		Status:             NormalizePaymentStatusOrPending(result.Status),
		OccurredAt:         eventTime,
	}
}

func (p *PaymentIntent) BuildRefundedEventData(result PaymentProviderResult, occurredAt time.Time) sharedevents.PaymentRefundedEvent {
	p.ensureWorkflowDefaults()
	eventTime := normalizePaymentTime(occurredAt)
	current := p.CurrentProviderResult(result)
	return sharedevents.PaymentRefundedEvent{
		Workflow:           p.Workflow,
		PaymentID:          p.TransactionID,
		TransactionID:      p.TransactionID,
		Provider:           p.Provider,
		ClearingAccountKey: p.ClearingAccountKey,
		DebitAccountID:     p.DebitAccountID,
		ProviderEventID:    strings.TrimSpace(current.EventID),
		ProviderEventType:  strings.TrimSpace(current.EventType),
		ProviderPaymentRef: coalescePaymentValue(current.ExternalRef, p.ExternalRef),
		Amount:             paymentResultAmountOrDefault(current.Amount, p.Amount),
		FeeAmount:          p.FeeAmount,
		ProviderAmount:     p.ProviderAmount,
		Currency:           p.Currency,
		CreditAccountID:    p.CreditAccountID,
		IdempotencyKey:     p.TransitionIdempotencyKey(sharedevents.EventPaymentRefunded),
		RefundedAt:         eventTime,
	}
}

func (p *PaymentIntent) BuildChargebackEventData(result PaymentProviderResult, occurredAt time.Time) sharedevents.PaymentChargebackEvent {
	p.ensureWorkflowDefaults()
	eventTime := normalizePaymentTime(occurredAt)
	current := p.CurrentProviderResult(result)
	return sharedevents.PaymentChargebackEvent{
		Workflow:           p.Workflow,
		PaymentID:          p.TransactionID,
		TransactionID:      p.TransactionID,
		Provider:           p.Provider,
		ClearingAccountKey: p.ClearingAccountKey,
		DebitAccountID:     p.DebitAccountID,
		ProviderEventID:    strings.TrimSpace(current.EventID),
		ProviderEventType:  strings.TrimSpace(current.EventType),
		ProviderPaymentRef: coalescePaymentValue(current.ExternalRef, p.ExternalRef),
		Amount:             paymentResultAmountOrDefault(current.Amount, p.Amount),
		FeeAmount:          p.FeeAmount,
		ProviderAmount:     p.ProviderAmount,
		Currency:           p.Currency,
		CreditAccountID:    p.CreditAccountID,
		IdempotencyKey:     p.TransitionIdempotencyKey(sharedevents.EventPaymentChargeback),
		ChargedBackAt:      eventTime,
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

func (p *PaymentIntent) NewProcessedTransitionEvent(eventName string, createdAt time.Time) (*ProcessedPaymentEvent, error) {
	return NewProcessedPaymentEvent(
		p.Provider,
		p.TransitionIdempotencyKey(eventName),
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

func paymentResultAmountOrDefault(amount int64, fallback int64) int64 {
	if amount != 0 {
		return amount
	}
	return fallback
}

func (p *PaymentIntent) ensureWorkflowDefaults() {
	if p == nil {
		return
	}
	p.TransactionID = strings.TrimSpace(p.TransactionID)
	if p.Workflow = NormalizePaymentWorkflow(p.Workflow); p.Workflow == "" {
		p.Workflow = PaymentWorkflowTopUp
	}
	p.Provider = strings.ToLower(strings.TrimSpace(p.Provider))
	p.ExternalRef = strings.TrimSpace(p.ExternalRef)
	p.DestinationAccountID = strings.TrimSpace(p.DestinationAccountID)
	p.Currency = strings.ToUpper(strings.TrimSpace(p.Currency))
	p.ClearingAccountKey = effectivePaymentClearingAccountKey(p.Provider, p.ClearingAccountKey)
	p.DebitAccountID = strings.TrimSpace(p.DebitAccountID)
	p.CreditAccountID = strings.TrimSpace(p.CreditAccountID)
	if p.ProviderAmount <= 0 {
		switch p.Workflow {
		case PaymentWorkflowWithdrawal:
			p.ProviderAmount = p.Amount
		default:
			p.ProviderAmount = p.Amount + p.FeeAmount
		}
	}
	if p.Status = NormalizePaymentStatus(p.Status); p.Status == "" {
		p.Status = PaymentStatusCreating
	}
}

func (p *PaymentIntent) CurrentProviderResult(source PaymentProviderResult) PaymentProviderResult {
	if p == nil {
		return PaymentProviderResult{}
	}

	amount := source.Amount
	if amount == 0 {
		amount = p.ProviderAmount
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
