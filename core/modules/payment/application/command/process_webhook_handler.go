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
			if err := intent.ApplyProviderResult(domainResult, time.Now().UTC()); err != nil {
				return stackErr.Error(err)
			}
			if err := tx.ProviderPaymentRepository().UpdateIntentProviderState(ctx, intent.TransactionID, intent.ExternalRef, intent.Status); err != nil {
				return stackErr.Error(err)
			}
			if intent.Status == entity.PaymentStatusFailed {
				return tx.ProviderPaymentRepository().AppendOutboxEvent(ctx, intent.FailedEvent(domainResult, time.Now().UTC()))
			}
			return nil
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
