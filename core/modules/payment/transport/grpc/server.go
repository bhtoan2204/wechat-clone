package grpc

import (
	"context"
	"errors"

	"wechat-clone/core/modules/payment/application/dto/in"
	"wechat-clone/core/modules/payment/application/dto/out"
	paymentservice "wechat-clone/core/modules/payment/application/service"
	"wechat-clone/core/shared/pkg/actorctx"
	"wechat-clone/core/shared/pkg/cqrs"
	paymentv1 "wechat-clone/core/shared/transport/grpc/gen/payment/v1"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const (
	metadataActorAccountID = "x-actor-account-id"
	metadataActorEmail     = "x-actor-email"
	metadataActorRole      = "x-actor-role"
)

type paymentGRPCServer struct {
	paymentv1.PaymentServiceServer
	createPayment  cqrs.Dispatcher[*in.CreatePaymentRequest, *out.CreatePaymentResponse]
	processWebhook cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse]
}

func NewServer(
	createPayment cqrs.Dispatcher[*in.CreatePaymentRequest, *out.CreatePaymentResponse],
	processWebhook cqrs.Dispatcher[*in.ProcessWebhookRequest, *out.ProcessWebhookResponse],
) paymentv1.PaymentServiceServer {
	return &paymentGRPCServer{
		createPayment:  createPayment,
		processWebhook: processWebhook,
	}
}

func (s *paymentGRPCServer) CreatePaymentIntent(ctx context.Context, req *paymentv1.CreatePaymentIntentRequest) (*paymentv1.CreatePaymentIntentResponse, error) {
	ctx = withActorFromMetadata(ctx)

	response, err := s.createPayment.Dispatch(ctx, &in.CreatePaymentRequest{
		Provider: req.GetProvider(),
		Amount:   req.GetAmount(),
		Currency: req.GetCurrency(),
		Metadata: req.GetMetadata(),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return &paymentv1.CreatePaymentIntentResponse{
		Provider:       response.Provider,
		Workflow:       response.Workflow,
		TransactionId:  response.TransactionID,
		ExternalRef:    response.ExternalRef,
		Amount:         response.Amount,
		FeeAmount:      response.FeeAmount,
		ProviderAmount: response.ProviderAmount,
		Status:         response.Status,
		CheckoutUrl:    response.CheckoutURL,
	}, nil
}

func (s *paymentGRPCServer) ProcessProviderWebhook(ctx context.Context, req *paymentv1.ProcessProviderWebhookRequest) (*paymentv1.ProcessProviderWebhookResponse, error) {
	response, err := s.processWebhook.Dispatch(ctx, &in.ProcessWebhookRequest{
		Provider:  req.GetProvider(),
		Signature: req.GetSignature(),
		Payload:   req.GetPayload(),
	})
	if err != nil {
		return nil, mapGRPCError(err)
	}

	return &paymentv1.ProcessProviderWebhookResponse{
		Provider:      response.Provider,
		TransactionId: response.TransactionID,
		ExternalRef:   response.ExternalRef,
		Status:        response.Status,
		Duplicate:     response.Duplicate,
		LedgerPosted:  response.LedgerPosted,
		Events:        paymentIntegrationEvents(response.Events),
	}, nil
}

func paymentIntegrationEvents(events []out.PaymentIntegrationEvent) []*paymentv1.PaymentIntegrationEvent {
	if len(events) == 0 {
		return nil
	}

	items := make([]*paymentv1.PaymentIntegrationEvent, 0, len(events))
	for _, event := range events {
		items = append(items, &paymentv1.PaymentIntegrationEvent{
			Name:     event.Name,
			DataJson: event.DataJson,
		})
	}
	return items
}

func withActorFromMetadata(ctx context.Context) context.Context {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ctx
	}

	accountID := firstMetadata(md, metadataActorAccountID)
	if accountID == "" {
		return ctx
	}

	return actorctx.WithActor(ctx, actorctx.Actor{
		AccountID: accountID,
		Email:     firstMetadata(md, metadataActorEmail),
		Role:      firstMetadata(md, metadataActorRole),
	})
}

func firstMetadata(md metadata.MD, key string) string {
	values := md.Get(key)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func mapGRPCError(err error) error {
	switch {
	case errors.Is(err, paymentservice.ErrValidation):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, paymentservice.ErrPaymentUnauthorized):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, paymentservice.ErrDuplicatePayment):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, paymentservice.ErrPaymentIntentNotFound):
		return status.Error(codes.NotFound, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}
