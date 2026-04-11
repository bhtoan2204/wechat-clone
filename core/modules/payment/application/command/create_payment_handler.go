package command

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	paymentservice "go-socket/core/modules/payment/application/service"
	"go-socket/core/modules/payment/domain/entity"
	repos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/modules/payment/providers"
	"go-socket/core/shared/pkg/cqrs"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"
	"go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type createPaymentHandler struct {
	baseRepo        repos.Repos
	providerService paymentservice.ProviderService
}

func NewCreatePayment(
	baseRepo repos.Repos,
	services paymentservice.Services,
) cqrs.Handler[*in.CreatePaymentRequest, *out.CreatePaymentResponse] {
	return &createPaymentHandler{
		baseRepo:        baseRepo,
		providerService: services.ProviderService(),
	}
}

func (u *createPaymentHandler) Handle(ctx context.Context, req *in.CreatePaymentRequest) (*out.CreatePaymentResponse, error) {
	log := logging.FromContext(ctx).Named("CreatePayment")
	if err := wrapValidation(req.Validate()); err != nil {
		return nil, stackErr.Error(err)
	}

	transactionID := uuid.New().String()
	provider, err := u.providerService.Get(req.Provider)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	now := time.Now().UTC()
	intent, err := entity.NewPaymentIntent(
		transactionID,
		req.Provider,
		req.Amount,
		req.Currency,
		req.DebitAccountID,
		req.CreditAccountID,
		now,
	)
	if err != nil {
		return nil, wrapValidation(err)
	}

	if err := u.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		return stackErr.Error(tx.ProviderPaymentRepository().CreatePaymentIntent(
			ctx,
			intent,
			intent.CreatedEvent(req.Metadata, now),
		))
	}); err != nil {
		if errors.Is(err, repos.ErrProviderPaymentDuplicateIntent) {
			return nil, fmt.Errorf("%v: %s", paymentservice.ErrDuplicatePayment, transactionID)
		}
		return nil, stackErr.Error(err)
	}

	response, err := provider.CreatePayment(ctx, providers.CreatePaymentRequest{
		TransactionID:   transactionID,
		Amount:          req.Amount,
		Currency:        req.Currency,
		DebitAccountID:  req.DebitAccountID,
		CreditAccountID: req.CreditAccountID,
		Metadata:        req.Metadata,
	})
	if err != nil {
		log.Errorw("provider create payment failed",
			"provider", provider.Name(),
			"transaction_id", transactionID,
			zap.Error(err),
		)
		if stateErr := intent.MarkCreateFailed(time.Now().UTC()); stateErr == nil {
			_ = u.baseRepo.ProviderPaymentRepository().SavePaymentIntent(ctx, intent)
		}
		return nil, stackErr.Error(err)
	}

	providerResult := entity.PaymentProviderResult{
		TransactionID: response.TransactionID,
		Status:        response.Status,
		ExternalRef:   response.ExternalRef,
	}
	targetStatus := entity.NormalizePaymentStatusOrPending(providerResult.Status)

	persistedIntent, err := u.baseRepo.ProviderPaymentRepository().GetIntentByTransactionID(ctx, transactionID)
	if err != nil {
		if errors.Is(err, repos.ErrProviderPaymentNotFound) {
			return nil, fmt.Errorf("%v: %s", paymentservice.ErrPaymentIntentNotFound, transactionID)
		}
		return nil, stackErr.Error(err)
	}

	if err := u.baseRepo.WithTransaction(ctx, func(tx repos.Repos) error {
		updatedAt := time.Now().UTC()
		if err := persistedIntent.ApplyProviderResult(providerResult, updatedAt); err != nil {
			return stackErr.Error(err)
		}

		outboxEvents := make([]eventpkg.Event, 0, 1)
		if response.CheckoutURL != "" || persistedIntent.ExternalRef != "" {
			outboxEvents = append(outboxEvents, persistedIntent.CheckoutSessionCreatedEvent(response.CheckoutURL, updatedAt))
		}

		if persistedIntent.Status == entity.PaymentStatusSuccess {
			return finalizeSuccessfulPaymentTx(
				ctx,
				tx.ProviderPaymentRepository(),
				persistedIntent,
				entity.PaymentProviderResult{
					TransactionID: response.TransactionID,
					Status:        persistedIntent.Status,
					Amount:        persistedIntent.Amount,
					Currency:      persistedIntent.Currency,
					ExternalRef:   persistedIntent.ExternalRef,
				},
				outboxEvents...,
			)
		}
		if persistedIntent.Status == entity.PaymentStatusFailed {
			outboxEvents = append(outboxEvents, persistedIntent.FailedEvent(entity.PaymentProviderResult{
				TransactionID: response.TransactionID,
				Status:        persistedIntent.Status,
				Amount:        persistedIntent.Amount,
				Currency:      persistedIntent.Currency,
				ExternalRef:   persistedIntent.ExternalRef,
			}, updatedAt))
		}
		return tx.ProviderPaymentRepository().SavePaymentIntent(ctx, persistedIntent, outboxEvents...)
	}); err != nil {
		return nil, stackErr.Error(err)
	}

	log.Infow("payment created",
		zap.String("provider", provider.Name()),
		zap.String("transaction_id", response.TransactionID),
		zap.String("status", targetStatus),
		zap.String("external_ref", response.ExternalRef),
	)

	return &out.CreatePaymentResponse{
		Provider:      strings.ToLower(provider.Name()),
		TransactionID: response.TransactionID,
		ExternalRef:   response.ExternalRef,
		Status:        targetStatus,
		CheckoutURL:   response.CheckoutURL,
	}, nil
}
