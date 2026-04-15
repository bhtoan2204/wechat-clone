package stripe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"go-socket/core/modules/payment/domain/entity"
	"go-socket/core/modules/payment/providers"
	"go-socket/core/shared/config"
	"go-socket/core/shared/pkg/stackErr"

	stripe "github.com/stripe/stripe-go/v75"
	stripeclient "github.com/stripe/stripe-go/v75/client"
	stripewebhook "github.com/stripe/stripe-go/v75/webhook"
)

var _ providers.PaymentProvider = (*Provider)(nil)

const (
	ProviderName = "stripe"
	apiVersion   = "2026-02-25.clover"
)

type Provider struct {
	secretKey     string
	webhookSecret string
	successURL    string
	cancelURL     string
	httpClient    *http.Client
	apiBaseURL    string
	client        *stripeclient.API
}

func NewProvider(cfg config.LedgerStripeConfig) *Provider {
	httpClient := &http.Client{
		Timeout: 15 * time.Second,
	}

	provider := &Provider{
		secretKey:     strings.TrimSpace(cfg.SecretKey),
		webhookSecret: strings.TrimSpace(cfg.WebhookSecret),
		successURL:    strings.TrimSpace(cfg.SuccessURL),
		cancelURL:     strings.TrimSpace(cfg.CancelURL),
		httpClient:    httpClient,
		apiBaseURL:    stripe.APIURL,
	}

	provider.client = newStripeClient(provider.secretKey, provider.httpClient, provider.apiBaseURL)
	return provider
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

	params := &stripe.CheckoutSessionParams{
		Params: stripe.Params{
			Context: ctx,
			Headers: http.Header{
				"Stripe-Version": []string{apiVersion},
			},
		},
		Mode:              stripe.String(string(stripe.CheckoutSessionModePayment)),
		SuccessURL:        stripe.String(successURL),
		CancelURL:         stripe.String(cancelURL),
		ClientReferenceID: stripe.String(req.TransactionID),
		Metadata: map[string]string{
			"transaction_id": req.TransactionID,
		},
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				Quantity: stripe.Int64(1),
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency:   stripe.String(strings.ToLower(req.Currency)),
					UnitAmount: stripe.Int64(req.Amount),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name: stripe.String(firstNonEmpty(req.Metadata["product_name"], "Wallet deposit")),
					},
				},
			},
		},
		PaymentIntentData: &stripe.CheckoutSessionPaymentIntentDataParams{
			Metadata: map[string]string{
				"transaction_id": req.TransactionID,
			},
		},
	}

	if email := strings.TrimSpace(req.Metadata["customer_email"]); email != "" {
		params.CustomerEmail = stripe.String(email)
	}
	if destination := strings.TrimSpace(req.Metadata["destination_account"]); destination != "" {
		params.PaymentIntentData.TransferData = &stripe.CheckoutSessionPaymentIntentDataTransferDataParams{
			Destination: stripe.String(destination),
		}
	}
	if onBehalfOf := strings.TrimSpace(req.Metadata["on_behalf_of"]); onBehalfOf != "" {
		params.PaymentIntentData.OnBehalfOf = stripe.String(onBehalfOf)
	}
	if applicationFee := strings.TrimSpace(req.Metadata["application_fee_amount"]); applicationFee != "" {
		feeAmount, err := strconv.ParseInt(applicationFee, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("invalid application_fee_amount: %v", err)
		}
		params.PaymentIntentData.ApplicationFeeAmount = stripe.Int64(feeAmount)
	}
	if statementDescriptor := strings.TrimSpace(req.Metadata["statement_descriptor"]); statementDescriptor != "" {
		params.PaymentIntentData.StatementDescriptorSuffix = stripe.String(statementDescriptor)
	}

	session, err := p.stripeClient().CheckoutSessions.New(params)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if session.ID == "" {
		return nil, fmt.Errorf("stripe checkout session response missing id")
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

	event, err := stripewebhook.ConstructEventWithOptions(payload, signature, p.webhookSecret, stripewebhook.ConstructEventOptions{
		IgnoreAPIVersionMismatch: true,
	})
	if err != nil {
		switch {
		case errors.Is(err, stripewebhook.ErrInvalidHeader),
			errors.Is(err, stripewebhook.ErrNoValidSignature),
			errors.Is(err, stripewebhook.ErrNotSigned),
			errors.Is(err, stripewebhook.ErrTooOld):
			return nil, providers.ErrInvalidWebhookSignature
		default:
			return nil, stackErr.Error(err)
		}
	}

	rawObject := ""
	if event.Data != nil {
		rawObject = string(event.Data.Raw)
	}

	return &providers.WebhookEvent{
		Provider:  ProviderName,
		EventID:   event.ID,
		EventType: string(event.Type),
		Attributes: map[string]string{
			"api_version": event.APIVersion,
			"object":      rawObject,
		},
	}, nil
}

func (p *Provider) ParseEvent(_ context.Context, event *providers.WebhookEvent) (*providers.PaymentResult, error) {
	switch stripe.EventType(event.EventType) {
	case stripe.EventTypeCheckoutSessionCompleted,
		stripe.EventTypeCheckoutSessionAsyncPaymentSucceeded,
		stripe.EventTypeCheckoutSessionAsyncPaymentFailed,
		stripe.EventTypeCheckoutSessionExpired:
		var session stripe.CheckoutSession
		if err := json.Unmarshal([]byte(event.Attributes["object"]), &session); err != nil {
			return nil, fmt.Errorf("decode stripe checkout session event: %v", err)
		}

		return &providers.PaymentResult{
			TransactionID: strings.TrimSpace(session.ClientReferenceID),
			EventID:       event.EventID,
			EventType:     event.EventType,
			Status:        stripeSessionStatus(event.EventType, string(session.PaymentStatus)),
			Amount:        session.AmountTotal,
			Currency:      string(session.Currency),
			ExternalRef:   session.ID,
		}, nil
	case stripe.EventTypePaymentIntentSucceeded,
		stripe.EventTypePaymentIntentPaymentFailed:
		var intent stripe.PaymentIntent
		if err := json.Unmarshal([]byte(event.Attributes["object"]), &intent); err != nil {
			return nil, fmt.Errorf("decode stripe payment intent event: %v", err)
		}

		return &providers.PaymentResult{
			TransactionID: strings.TrimSpace(intent.Metadata["transaction_id"]),
			EventID:       event.EventID,
			EventType:     event.EventType,
			Status:        stripePaymentIntentStatus(event.EventType, string(intent.Status)),
			Amount:        intent.Amount,
			Currency:      string(intent.Currency),
			ExternalRef:   intent.ID,
		}, nil
	case stripe.EventTypeChargeSucceeded, stripe.EventTypeChargeFailed:
		var charge stripe.Charge
		if err := json.Unmarshal([]byte(event.Attributes["object"]), &charge); err != nil {
			return nil, fmt.Errorf("decode stripe charge event: %v", err)
		}

		return &providers.PaymentResult{
			TransactionID: stripeChargeTransactionID(&charge),
			EventID:       event.EventID,
			EventType:     event.EventType,
			Status:        stripeChargeStatus(event.EventType, string(charge.Status), charge.Paid),
			Amount:        charge.Amount,
			Currency:      string(charge.Currency),
			ExternalRef:   charge.ID,
		}, nil
	default:
		return nil, providers.ErrWebhookEventIgnored
	}
}

func (p *Provider) stripeClient() *stripeclient.API {
	if p.client == nil {
		p.client = newStripeClient(p.secretKey, p.httpClient, p.apiBaseURL)
	}
	return p.client
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

func stripeChargeTransactionID(charge *stripe.Charge) string {
	if charge == nil {
		return ""
	}
	if transactionID := strings.TrimSpace(charge.Metadata["transaction_id"]); transactionID != "" {
		return transactionID
	}
	if charge.PaymentIntent != nil {
		return strings.TrimSpace(charge.PaymentIntent.Metadata["transaction_id"])
	}
	return ""
}

func stripeChargeStatus(eventType, status string, paid bool) string {
	switch eventType {
	case "charge.succeeded":
		return entity.PaymentStatusSuccess
	case "charge.failed":
		return entity.PaymentStatusFailed
	}

	switch strings.ToLower(strings.TrimSpace(status)) {
	case "succeeded":
		return entity.PaymentStatusSuccess
	case "failed":
		return entity.PaymentStatusFailed
	}

	if paid {
		return entity.PaymentStatusSuccess
	}
	return entity.PaymentStatusPending
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func newStripeClient(secretKey string, httpClient *http.Client, apiBaseURL string) *stripeclient.API {
	if httpClient == nil {
		httpClient = &http.Client{
			Timeout: 15 * time.Second,
		}
	}

	baseURL := strings.TrimRight(strings.TrimSpace(apiBaseURL), "/")
	if baseURL == "" {
		baseURL = stripe.APIURL
	}

	backends := &stripe.Backends{
		API: stripe.GetBackendWithConfig(stripe.APIBackend, &stripe.BackendConfig{
			HTTPClient: httpClient,
			URL:        stripe.String(baseURL),
		}),
		Connect: stripe.GetBackendWithConfig(stripe.ConnectBackend, &stripe.BackendConfig{
			HTTPClient: httpClient,
		}),
		Uploads: stripe.GetBackendWithConfig(stripe.UploadsBackend, &stripe.BackendConfig{
			HTTPClient: httpClient,
		}),
	}

	return stripeclient.New(secretKey, backends)
}
