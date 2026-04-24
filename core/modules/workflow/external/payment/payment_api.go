package payment

import (
	"context"
	"time"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/infra/tracing"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"
	paymentv1 "wechat-clone/core/shared/transport/grpc/gen/payment/v1"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
)

type PaymentGrpc interface {
	paymentv1.PaymentServiceClient
	Close() error
}

type paymentGrpc struct {
	cc         *grpc.ClientConn
	grpcClient paymentv1.PaymentServiceClient
}

func New(ctx context.Context, cfg *config.Config) (PaymentGrpc, error) {
	log := logging.FromContext(ctx)
	cc, err := grpc.NewClient(
		cfg.GrpcConfig.PaymentGRPCBaseURL,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithChainUnaryInterceptor(),
		grpc.WithStatsHandler(tracing.GrpcStatsHandler()),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                10 * time.Second,
			Timeout:             3 * time.Second,
			PermitWithoutStream: true,
		}),
	)
	if err != nil {
		log.Warnw("Could not setup Payment Service GRPC connection", zap.String("connection string", cfg.GrpcConfig.PaymentGRPCBaseURL), zap.Error(err))
		return nil, stackErr.Error(err)
	}

	return paymentGrpc{
		cc:         cc,
		grpcClient: paymentv1.NewPaymentServiceClient(cc),
	}, nil
}

func (p paymentGrpc) CreatePaymentIntent(ctx context.Context, req *paymentv1.CreatePaymentIntentRequest, opts ...grpc.CallOption) (*paymentv1.CreatePaymentIntentResponse, error) {
	response, err := p.grpcClient.CreatePaymentIntent(ctx, req, opts...)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return response, nil
}

func (p paymentGrpc) ProcessProviderWebhook(ctx context.Context, req *paymentv1.ProcessProviderWebhookRequest, opts ...grpc.CallOption) (*paymentv1.ProcessProviderWebhookResponse, error) {
	response, err := p.grpcClient.ProcessProviderWebhook(ctx, req, opts...)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return response, nil
}

func (p paymentGrpc) Close() error {
	if p.cc == nil {
		return nil
	}
	return stackErr.Error(p.cc.Close())
}
