package mock

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"go-socket/core/modules/ledger/domain/entity"
	"go-socket/core/modules/ledger/providers"
)

const ProviderName = "mock"

type Provider struct {
	webhookSecret string
}

func NewProvider(webhookSecret string) *Provider {
	return &Provider{webhookSecret: webhookSecret}
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) CreatePayment(_ context.Context, req providers.CreatePaymentRequest) (*providers.CreatePaymentResponse, error) {
	status := entity.PaymentStatusPending
	if strings.EqualFold(req.Metadata["auto_capture"], "true") {
		status = entity.PaymentStatusSuccess
	}

	externalRef := fmt.Sprintf("mock_%s", req.TransactionID)
	return &providers.CreatePaymentResponse{
		Provider:      ProviderName,
		TransactionID: req.TransactionID,
		ExternalRef:   externalRef,
		Status:        status,
		CheckoutURL:   fmt.Sprintf("https://mock-payments.local/checkout/%s", externalRef),
	}, nil
}

func (p *Provider) VerifyWebhook(_ context.Context, payload []byte, signature string) (*providers.WebhookEvent, error) {
	if strings.TrimSpace(signature) != p.webhookSecret {
		return nil, providers.ErrInvalidWebhookSignature
	}

	var body webhookPayload
	if err := json.Unmarshal(payload, &body); err != nil {
		return nil, fmt.Errorf("decode mock webhook payload: %w", err)
	}

	return &providers.WebhookEvent{
		Provider:  ProviderName,
		EventID:   body.EventID,
		EventType: body.EventType,
		Attributes: map[string]string{
			"transaction_id": body.TransactionID,
			"external_ref":   body.ExternalRef,
			"status":         body.Status,
			"amount":         fmt.Sprintf("%d", body.Amount),
			"currency":       body.Currency,
		},
	}, nil
}

func (p *Provider) ParseEvent(_ context.Context, event *providers.WebhookEvent) (*providers.PaymentResult, error) {
	amount := int64(0)
	if rawAmount := strings.TrimSpace(event.Attributes["amount"]); rawAmount != "" {
		if _, err := fmt.Sscanf(rawAmount, "%d", &amount); err != nil {
			return nil, fmt.Errorf("parse mock webhook amount: %w", err)
		}
	}

	return &providers.PaymentResult{
		TransactionID: strings.TrimSpace(event.Attributes["transaction_id"]),
		Status:        strings.TrimSpace(event.Attributes["status"]),
		Amount:        amount,
		Currency:      strings.TrimSpace(event.Attributes["currency"]),
		ExternalRef:   strings.TrimSpace(event.Attributes["external_ref"]),
	}, nil
}

type webhookPayload struct {
	EventID       string `json:"event_id"`
	EventType     string `json:"event_type"`
	TransactionID string `json:"transaction_id"`
	ExternalRef   string `json:"external_ref"`
	Status        string `json:"status"`
	Amount        int64  `json:"amount"`
	Currency      string `json:"currency"`
}
