package command

import (
	"context"

	paymentin "go-socket/core/modules/payment/application/dto/in"
	paymentout "go-socket/core/modules/payment/application/dto/out"
	paymentservice "go-socket/core/modules/payment/application/service"
	"go-socket/core/shared/pkg/cqrs"
)

type createProviderPaymentHandler struct {
	service *paymentservice.PaymentService
}

func NewCreateProviderPaymentHandler(service *paymentservice.PaymentService) cqrs.Handler[*paymentin.CreatePaymentRequest, *paymentout.CreatePaymentResponse] {
	return &createProviderPaymentHandler{service: service}
}

func (h *createProviderPaymentHandler) Handle(ctx context.Context, req *paymentin.CreatePaymentRequest) (*paymentout.CreatePaymentResponse, error) {
	return h.service.CreatePayment(ctx, req)
}
