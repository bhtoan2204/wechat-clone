package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"go-socket/core/modules/ledger/domain/entity"
	"go-socket/core/modules/ledger/providers"
	"go-socket/core/shared/config"
)

const (
	ProviderName = "stripe"
	apiVersion   = "2026-02-25.clover"
	apiBaseURL   = "https://api.stripe.com"
)

type Provider struct {
	secretKey     string
	webhookSecret string
	successURL    string
	cancelURL     string
	httpClient    *http.Client
}

func NewProvider(cfg config.LedgerStripeConfig) *Provider {
	return &Provider{
		secretKey:     strings.TrimSpace(cfg.SecretKey),
		webhookSecret: strings.TrimSpace(cfg.WebhookSecret),
		successURL:    strings.TrimSpace(cfg.SuccessURL),
		cancelURL:     strings.TrimSpace(cfg.CancelURL),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (p *Provider) Name() string {
	return ProviderName
}

func (p *Provider) Enabled() bool {
	return p.secretKey != ""
}

func (p *Provider) CreatePayment(ctx context.Context, req providers.CreatePaymentRequest) (*providers.CreatePaymentResponse, error) {
	if !p.Enabled() {
		return nil, fmt.Errorf("stripe provider is not configured")
	}

	successURL := firstNonEmpty(req.Metadata["success_url"], p.successURL)
	cancelURL := firstNonEmpty(req.Metadata["cancel_url"], p.cancelURL)
	if successURL == "" || cancelURL == "" {
		return nil, fmt.Errorf("stripe success_url and cancel_url are required")
	}

	form := url.Values{}
	form.Set("mode", "payment")
	form.Set("success_url", successURL)
	form.Set("cancel_url", cancelURL)
	form.Set("client_reference_id", req.TransactionID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("line_items[0][price_data][currency]", strings.ToLower(req.Currency))
	form.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(req.Amount, 10))
	form.Set("line_items[0][price_data][product_data][name]", firstNonEmpty(req.Metadata["product_name"], "Wallet deposit"))
	form.Set("metadata[transaction_id]", req.TransactionID)
	form.Set("metadata[debit_account_id]", req.DebitAccountID)
	form.Set("metadata[credit_account_id]", req.CreditAccountID)

	if email := strings.TrimSpace(req.Metadata["customer_email"]); email != "" {
		form.Set("customer_email", email)
	}
	if destination := strings.TrimSpace(req.Metadata["destination_account"]); destination != "" {
		form.Set("payment_intent_data[transfer_data][destination]", destination)
	}
	if onBehalfOf := strings.TrimSpace(req.Metadata["on_behalf_of"]); onBehalfOf != "" {
		form.Set("payment_intent_data[on_behalf_of]", onBehalfOf)
	}
	if applicationFee := strings.TrimSpace(req.Metadata["application_fee_amount"]); applicationFee != "" {
		form.Set("payment_intent_data[application_fee_amount]", applicationFee)
	}
	if statementDescriptor := strings.TrimSpace(req.Metadata["statement_descriptor"]); statementDescriptor != "" {
		form.Set("payment_intent_data[statement_descriptor_suffix]", statementDescriptor)
	}

	respBody, statusCode, err := p.doFormRequest(ctx, http.MethodPost, apiBaseURL+"/v1/checkout/sessions", form)
	if err != nil {
		return nil, err
	}

	var session checkoutSession
	if err := json.Unmarshal(respBody, &session); err != nil {
		return nil, fmt.Errorf("decode stripe checkout session: %w", err)
	}
	if session.ID == "" {
		return nil, fmt.Errorf("stripe checkout session response missing id: status=%d", statusCode)
	}

	return &providers.CreatePaymentResponse{
		Provider:      ProviderName,
		TransactionID: req.TransactionID,
		ExternalRef:   session.ID,
		Status:        entity.PaymentStatusPending,
		CheckoutURL:   session.URL,
	}, nil
}

func (p *Provider) VerifyWebhook(_ context.Context, payload []byte, signature string) (*providers.WebhookEvent, error) {
	if strings.TrimSpace(p.webhookSecret) == "" {
		return nil, fmt.Errorf("stripe webhook secret is not configured")
	}

	timestamp, expected, err := parseStripeSignature(signature)
	if err != nil {
		return nil, providers.ErrInvalidWebhookSignature
	}

	mac := hmac.New(sha256.New, []byte(p.webhookSecret))
	_, _ = mac.Write([]byte(timestamp))
	_, _ = mac.Write([]byte("."))
	_, _ = mac.Write(payload)
	actual := hex.EncodeToString(mac.Sum(nil))
	if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) != 1 {
		return nil, providers.ErrInvalidWebhookSignature
	}

	var envelope stripeEventEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("decode stripe webhook payload: %w", err)
	}

	return &providers.WebhookEvent{
		Provider:  ProviderName,
		EventID:   envelope.ID,
		EventType: envelope.Type,
		Attributes: map[string]string{
			"object": string(envelope.Data.Object),
		},
	}, nil
}

func (p *Provider) ParseEvent(_ context.Context, event *providers.WebhookEvent) (*providers.PaymentResult, error) {
	switch event.EventType {
	case "checkout.session.completed", "checkout.session.async_payment_succeeded", "checkout.session.async_payment_failed", "checkout.session.expired":
		var session checkoutSession
		if err := json.Unmarshal([]byte(event.Attributes["object"]), &session); err != nil {
			return nil, fmt.Errorf("decode stripe checkout session event: %w", err)
		}

		return &providers.PaymentResult{
			TransactionID: strings.TrimSpace(session.ClientReferenceID),
			Status:        stripeSessionStatus(event.EventType, session.PaymentStatus),
			Amount:        session.AmountTotal,
			Currency:      session.Currency,
			ExternalRef:   session.ID,
		}, nil
	case "payment_intent.succeeded", "payment_intent.payment_failed":
		var intent paymentIntentObject
		if err := json.Unmarshal([]byte(event.Attributes["object"]), &intent); err != nil {
			return nil, fmt.Errorf("decode stripe payment intent event: %w", err)
		}

		return &providers.PaymentResult{
			TransactionID: strings.TrimSpace(intent.Metadata.TransactionID),
			Status:        stripePaymentIntentStatus(event.EventType, intent.Status),
			Amount:        intent.Amount,
			Currency:      intent.Currency,
			ExternalRef:   intent.ID,
		}, nil
	default:
		return nil, fmt.Errorf("unsupported stripe event type: %s", event.EventType)
	}
}

func (p *Provider) doFormRequest(ctx context.Context, method, endpoint string, form url.Values) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, method, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, 0, fmt.Errorf("build stripe request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.secretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Stripe-Version", apiVersion)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("call stripe api: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read stripe response: %w", err)
	}

	if resp.StatusCode >= http.StatusBadRequest {
		return nil, resp.StatusCode, fmt.Errorf("stripe api error: status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return body, resp.StatusCode, nil
}

func parseStripeSignature(signature string) (string, string, error) {
	var timestamp string
	var v1 string

	for _, part := range strings.Split(signature, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			timestamp = value
		case "v1":
			v1 = value
		}
	}

	if timestamp == "" || v1 == "" {
		return "", "", fmt.Errorf("invalid stripe signature")
	}

	return timestamp, v1, nil
}

func stripeSessionStatus(eventType, paymentStatus string) string {
	switch eventType {
	case "checkout.session.async_payment_failed", "checkout.session.expired":
		return entity.PaymentStatusFailed
	}
	if strings.EqualFold(strings.TrimSpace(paymentStatus), "paid") {
		return entity.PaymentStatusSuccess
	}
	return entity.PaymentStatusPending
}

func stripePaymentIntentStatus(eventType, status string) string {
	switch eventType {
	case "payment_intent.succeeded":
		return entity.PaymentStatusSuccess
	case "payment_intent.payment_failed":
		return entity.PaymentStatusFailed
	}

	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded":
		return entity.PaymentStatusSuccess
	case "requires_payment_method", "canceled":
		return entity.PaymentStatusFailed
	default:
		return entity.PaymentStatusPending
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

type stripeEventEnvelope struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type checkoutSession struct {
	ID                string `json:"id"`
	URL               string `json:"url"`
	ClientReferenceID string `json:"client_reference_id"`
	PaymentStatus     string `json:"payment_status"`
	AmountTotal       int64  `json:"amount_total"`
	Currency          string `json:"currency"`
}

type paymentIntentObject struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Amount   int64  `json:"amount"`
	Currency string `json:"currency"`
	Metadata struct {
		TransactionID string `json:"transaction_id"`
	} `json:"metadata"`
}
