package temporal

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	workflowservice "wechat-clone/core/modules/workflow/application/service"
	workflowledger "wechat-clone/core/modules/workflow/external/ledger"
	workflowpayment "wechat-clone/core/modules/workflow/external/payment"
	"wechat-clone/core/shared/pkg/stackErr"
	ledgerv1 "wechat-clone/core/shared/transport/grpc/gen/ledger/v1"
	paymentv1 "wechat-clone/core/shared/transport/grpc/gen/payment/v1"

	"github.com/google/uuid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	temporalworkflow "go.temporal.io/sdk/workflow"
	"google.golang.org/grpc/metadata"
)

const (
	StripeTopUpTaskQueue                = "workflow:stripe:top-up"
	StripeTopUpWorkflowName             = "workflow.stripe.top_up.create"
	StripeWebhookWorkflowName           = "workflow.stripe.webhook.process"
	StripeCreateTopUpIntentActivityName = "workflow.stripe.top_up.create_intent"
	StripeProcessWebhookActivityName    = "workflow.stripe.webhook.process_payment"
	StripeApplyPaymentEventActivityName = "workflow.stripe.webhook.apply_ledger_event"
	metadataActorAccountID              = "x-actor-account-id"
	metadataActorEmail                  = "x-actor-email"
	metadataActorRole                   = "x-actor-role"
)

type stripeTopUpRunner struct {
	client client.Client
}

type stripeTopUpActivities struct {
	payment workflowpayment.PaymentGrpc
	ledger  workflowledger.LedgerGrpc
}

type workerRuntime struct {
	worker  worker.Worker
	closers []interface{ Close() error }
}

func NewStripeTopUpRunner(temporalClient client.Client) workflowservice.StripeTopUpWorkflowRunner {
	if temporalClient == nil {
		return nil
	}
	return &stripeTopUpRunner{client: temporalClient}
}

func NewWorkerRuntime(
	temporalClient client.Client,
	payment workflowpayment.PaymentGrpc,
	ledger workflowledger.LedgerGrpc,
) *workerRuntime {
	if temporalClient == nil {
		return &workerRuntime{}
	}

	w := worker.New(temporalClient, StripeTopUpTaskQueue, worker.Options{})
	activities := &stripeTopUpActivities{
		payment: payment,
		ledger:  ledger,
	}

	w.RegisterWorkflowWithOptions(StripeTopUpWorkflow, temporalworkflow.RegisterOptions{Name: StripeTopUpWorkflowName})
	w.RegisterWorkflowWithOptions(StripeWebhookWorkflow, temporalworkflow.RegisterOptions{Name: StripeWebhookWorkflowName})
	w.RegisterActivityWithOptions(activities.CreateTopUpIntent, activity.RegisterOptions{Name: StripeCreateTopUpIntentActivityName})
	w.RegisterActivityWithOptions(activities.ProcessWebhook, activity.RegisterOptions{Name: StripeProcessWebhookActivityName})
	w.RegisterActivityWithOptions(activities.ApplyPaymentEvent, activity.RegisterOptions{Name: StripeApplyPaymentEventActivityName})

	closers := make([]interface{ Close() error }, 0, 2)
	if payment != nil {
		closers = append(closers, payment)
	}
	if ledger != nil {
		closers = append(closers, ledger)
	}

	return &workerRuntime{worker: w, closers: closers}
}

func (r *stripeTopUpRunner) CreateStripeTopUp(ctx context.Context, input workflowservice.CreateStripeTopUpWorkflowInput) (*workflowservice.StripeTopUpWorkflowResult, error) {
	if r == nil || r.client == nil {
		return nil, stackErr.Error(workflowservice.ErrWorkflowUnavailable)
	}

	run, err := r.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                  fmt.Sprintf("stripe-top-up:create:%s", uuid.NewString()),
		TaskQueue:           StripeTopUpTaskQueue,
		WorkflowRunTimeout:  5 * time.Minute,
		WorkflowTaskTimeout: 15 * time.Second,
	}, StripeTopUpWorkflowName, input)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var result workflowservice.StripeTopUpWorkflowResult
	if err := run.Get(ctx, &result); err != nil {
		return nil, stackErr.Error(err)
	}
	return &result, nil
}

func (r *stripeTopUpRunner) ProcessStripeWebhook(ctx context.Context, input workflowservice.ProcessStripeWebhookWorkflowInput) (*workflowservice.StripeWebhookWorkflowResult, error) {
	if r == nil || r.client == nil {
		return nil, stackErr.Error(workflowservice.ErrWorkflowUnavailable)
	}

	run, err := r.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                  fmt.Sprintf("stripe-top-up:webhook:%s", uuid.NewString()),
		TaskQueue:           StripeTopUpTaskQueue,
		WorkflowRunTimeout:  5 * time.Minute,
		WorkflowTaskTimeout: 15 * time.Second,
	}, StripeWebhookWorkflowName, input)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var result workflowservice.StripeWebhookWorkflowResult
	if err := run.Get(ctx, &result); err != nil {
		return nil, stackErr.Error(err)
	}
	return &result, nil
}

func StripeTopUpWorkflow(ctx temporalworkflow.Context, input workflowservice.CreateStripeTopUpWorkflowInput) (*workflowservice.StripeTopUpWorkflowResult, error) {
	ctx = temporalworkflow.WithActivityOptions(ctx, stripeActivityOptions())

	var result workflowservice.StripeTopUpWorkflowResult
	err := temporalworkflow.ExecuteActivity(ctx, StripeCreateTopUpIntentActivityName, input).Get(ctx, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func StripeWebhookWorkflow(ctx temporalworkflow.Context, input workflowservice.ProcessStripeWebhookWorkflowInput) (*workflowservice.StripeWebhookWorkflowResult, error) {
	ctx = temporalworkflow.WithActivityOptions(ctx, stripeActivityOptions())

	var result workflowservice.StripeWebhookWorkflowResult
	if err := temporalworkflow.ExecuteActivity(ctx, StripeProcessWebhookActivityName, input).Get(ctx, &result); err != nil {
		return nil, err
	}

	for _, event := range result.Events {
		var applied bool
		if err := temporalworkflow.ExecuteActivity(ctx, StripeApplyPaymentEventActivityName, event).Get(ctx, &applied); err != nil {
			return nil, err
		}
		result.LedgerPosted = result.LedgerPosted || applied
	}
	result.Events = nil

	return &result, nil
}

func stripeActivityOptions() temporalworkflow.ActivityOptions {
	return temporalworkflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    10,
		},
	}
}

func (a *stripeTopUpActivities) CreateTopUpIntent(ctx context.Context, input workflowservice.CreateStripeTopUpWorkflowInput) (*workflowservice.StripeTopUpWorkflowResult, error) {
	if a == nil || a.payment == nil {
		return nil, stackErr.Error(errors.New("payment grpc client is unavailable"))
	}

	response, err := a.payment.CreatePaymentIntent(
		actorOutgoingContext(ctx, input.Actor),
		&paymentv1.CreatePaymentIntentRequest{
			Provider: "stripe",
			Amount:   input.Amount,
			Currency: input.Currency,
			Metadata: input.Metadata,
		},
	)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &workflowservice.StripeTopUpWorkflowResult{
		Provider:       response.GetProvider(),
		Workflow:       response.GetWorkflow(),
		TransactionID:  response.GetTransactionId(),
		ExternalRef:    response.GetExternalRef(),
		Amount:         response.GetAmount(),
		FeeAmount:      response.GetFeeAmount(),
		ProviderAmount: response.GetProviderAmount(),
		Status:         response.GetStatus(),
		CheckoutURL:    response.GetCheckoutUrl(),
	}, nil
}

func (a *stripeTopUpActivities) ProcessWebhook(ctx context.Context, input workflowservice.ProcessStripeWebhookWorkflowInput) (*workflowservice.StripeWebhookWorkflowResult, error) {
	if a == nil || a.payment == nil {
		return nil, stackErr.Error(errors.New("payment grpc client is unavailable"))
	}

	response, err := a.payment.ProcessProviderWebhook(ctx, &paymentv1.ProcessProviderWebhookRequest{
		Provider:  "stripe",
		Signature: input.Signature,
		Payload:   input.Payload,
	})
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &workflowservice.StripeWebhookWorkflowResult{
		Provider:      response.GetProvider(),
		TransactionID: response.GetTransactionId(),
		ExternalRef:   response.GetExternalRef(),
		Status:        response.GetStatus(),
		Duplicate:     response.GetDuplicate(),
		LedgerPosted:  response.GetLedgerPosted(),
		Events:        paymentIntegrationEvents(response.GetEvents()),
	}, nil
}

func (a *stripeTopUpActivities) ApplyPaymentEvent(ctx context.Context, event workflowservice.PaymentIntegrationEvent) (bool, error) {
	if a == nil || a.ledger == nil {
		return false, stackErr.Error(errors.New("ledger grpc client is unavailable"))
	}
	if strings.TrimSpace(event.Name) == "" {
		return false, nil
	}

	response, err := a.ledger.ApplyPaymentEvent(ctx, &ledgerv1.ApplyPaymentEventRequest{
		EventName:     event.Name,
		EventDataJson: event.DataJSON,
	})
	if err != nil {
		return false, stackErr.Error(err)
	}
	return response.GetApplied(), nil
}

func actorOutgoingContext(ctx context.Context, actor workflowservice.WorkflowActor) context.Context {
	pairs := make([]string, 0, 6)
	if strings.TrimSpace(actor.AccountID) != "" {
		pairs = append(pairs, metadataActorAccountID, strings.TrimSpace(actor.AccountID))
	}
	if strings.TrimSpace(actor.Email) != "" {
		pairs = append(pairs, metadataActorEmail, strings.TrimSpace(actor.Email))
	}
	if strings.TrimSpace(actor.Role) != "" {
		pairs = append(pairs, metadataActorRole, strings.TrimSpace(actor.Role))
	}
	if len(pairs) == 0 {
		return ctx
	}
	return metadata.NewOutgoingContext(ctx, metadata.Pairs(pairs...))
}

func paymentIntegrationEvents(events []*paymentv1.PaymentIntegrationEvent) []workflowservice.PaymentIntegrationEvent {
	if len(events) == 0 {
		return nil
	}

	items := make([]workflowservice.PaymentIntegrationEvent, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		items = append(items, workflowservice.PaymentIntegrationEvent{
			Name:     event.GetName(),
			DataJSON: event.GetDataJson(),
		})
	}
	return items
}

func (r *workerRuntime) Start() error {
	if r == nil || r.worker == nil {
		return nil
	}
	return stackErr.Error(r.worker.Start())
}

func (r *workerRuntime) Stop() error {
	if r == nil {
		return nil
	}
	if r.worker != nil {
		r.worker.Stop()
	}
	for _, closer := range r.closers {
		if closer == nil {
			continue
		}
		if err := closer.Close(); err != nil {
			return stackErr.Error(err)
		}
	}
	return nil
}
