package command

import (
	"context"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	paymentservice "go-socket/core/modules/payment/application/service"
	"go-socket/core/shared/pkg/cqrs"
)

type processWebhookHandler struct {
	paymentCommandService paymentservice.PaymentCommandService
}

func NewProcessWebhook(paymentCommandService paymentservice.PaymentCommandService) cqrs.Handler[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse] {
	return &processWebhookHandler{
		paymentCommandService: paymentCommandService,
	}
}

func (u *processWebhookHandler) Handle(ctx context.Context, req *in.ProcessWebhookRequest) (*out.ProcessWebhookResponse, error) {
	return u.paymentCommandService.ProcessWebhook(ctx, req)
}
