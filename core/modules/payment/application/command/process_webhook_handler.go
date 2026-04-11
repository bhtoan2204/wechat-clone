package command

import (
	"context"
	"time"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	paymentservice "go-socket/core/modules/payment/application/service"
	"go-socket/core/modules/payment/domain/entity"
	repos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/stackErr"
)

type processWebhookHandler struct {
	baseRepo        repos.Repos
	providerService paymentservice.ProviderService
}

func NewProcessWebhook(
	baseRepo repos.Repos,
	services paymentservice.Services,
) cqrs.Handler[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse] {
	return &processWebhookHandler{
		baseRepo:        baseRepo,
		providerService: services.ProviderService(),
	}
}

func (u *processWebhookHandler) Handle(ctx context.Context, req *in.ProcessWebhookRequest) (*out.ProcessWebhookResponse, error) {
	provider, err := u.providerService.Get(req.Provider)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	event, err := provider.VerifyWebhook(ctx, []byte(req.Payload), req.Signature)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	result, err := provider.ParseEvent(ctx, event)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	domainResult := toDomainPaymentResult(result)
	result.Status = domainResult.Status

	intent, err := findIntent(ctx, u.baseRepo.ProviderPaymentRepository(), provider.Name(), result)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if err := wrapValidation(intent.ValidateProviderResult(domainResult.Amount, domainResult.Currency)); err != nil {
		return nil, stackErr.Error(err)
	}

	if domainResult.Status != entity.PaymentStatusSuccess {
		if err := u.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
			updatedAt := time.Now().UTC()
			if err := intent.ApplyProviderResult(domainResult, updatedAt); err != nil {
				return stackErr.Error(err)
			}
			outboxEvents := make([]eventpkg.Event, 0, 1)
			if intent.Status == entity.PaymentStatusFailed {
				outboxEvents = append(outboxEvents, intent.FailedEvent(domainResult, updatedAt))
			}
			return tx.ProviderPaymentRepository().SavePaymentIntent(ctx, intent, outboxEvents...)
		}); err != nil {
			return nil, stackErr.Error(err)
		}
		return &out.ProcessWebhookResponse{
			Provider:      intent.Provider,
			TransactionID: intent.TransactionID,
			ExternalRef:   coalesce(result.ExternalRef, intent.ExternalRef),
			Status:        domainResult.Status,
		}, nil
	}

	duplicate, err := finalizeSuccessfulPayment(ctx, u.baseRepo, u.baseRepo.ProviderPaymentRepository(), intent, domainResult)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return &out.ProcessWebhookResponse{
		Provider:      intent.Provider,
		TransactionID: intent.TransactionID,
		ExternalRef:   coalesce(result.ExternalRef, intent.ExternalRef),
		Status:        entity.PaymentStatusSuccess,
		Duplicate:     duplicate,
		LedgerPosted:  false,
	}, nil
}
