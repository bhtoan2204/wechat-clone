package command

import (
	"context"

	paymentin "go-socket/core/modules/payment/application/dto/in"
	paymentout "go-socket/core/modules/payment/application/dto/out"
	paymentservice "go-socket/core/modules/payment/application/service"
	"go-socket/core/shared/pkg/cqrs"
)

type processProviderWebhookHandler struct {
	service *paymentservice.PaymentService
}

func NewProcessProviderWebhookHandler(service *paymentservice.PaymentService) cqrs.Handler[*paymentin.ProcessWebhookRequest, *paymentout.ProcessWebhookResponse] {
	return &processProviderWebhookHandler{service: service}
}

func (h *processProviderWebhookHandler) Handle(ctx context.Context, req *paymentin.ProcessWebhookRequest) (*paymentout.ProcessWebhookResponse, error) {
	return h.service.HandleWebhook(ctx, req.Provider, []byte(req.Payload), req.Signature)
}
