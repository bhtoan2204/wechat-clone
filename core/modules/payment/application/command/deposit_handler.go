package command

import (
	"context"
	"errors"
	"reflect"
	"time"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/domain/aggregate"
	"go-socket/core/modules/payment/domain/repos"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/shared/infra/xpaseto"
	eventpkg "go-socket/core/shared/pkg/event"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
)

type depositHandler struct {
	outboxRepo paymentrepos.PaymentOutboxEventsRepository
}

func NewDepositHandler(repos repos.Repos) DepositHandler {
	return &depositHandler{
		outboxRepo: repos.PaymentOutboxEventsRepository(),
	}
}

func (h *depositHandler) Handle(ctx context.Context, req *in.DepositRequest) (*out.DepositResponse, error) {
	log := logging.FromContext(ctx).Named("Deposit")

	if req == nil {
		err := errors.New("deposit request is nil")
		log.Errorw("Invalid deposit request", "error", err)
		return nil, stackerr.Error(err)
	}
	if err := req.Validate(); err != nil {
		log.Errorw("Invalid deposit request", "error", err)
		return nil, stackerr.Error(err)
	}

	account, ok := ctx.Value("account").(*xpaseto.PasetoPayload)
	if !ok || account == nil || account.AccountID == "" {
		err := errors.New("account not found")
		log.Errorw("Account not found", "error", err)
		return nil, stackerr.Error(err)
	}

	if h.outboxRepo == nil {
		err := errors.New("payment outbox repository is nil")
		log.Errorw("Outbox repository not initialized", "error", err)
		return nil, stackerr.Error(err)
	}

	now := time.Now().UTC()
	transactionID := uuid.NewString()

	agg := &aggregate.PaymentTransactionAggregate{}
	aggType := reflect.TypeOf(agg).Elem().Name()
	agg.SetAggregateType(aggType)
	if err := agg.SetID(transactionID); err != nil {
		log.Errorw("Failed to set aggregate id", "error", err)
		return nil, stackerr.Error(err)
	}

	if err := agg.ApplyChange(agg, &aggregate.EventPaymentTransactionDeposited{
		PaymentTransactionID:         transactionID,
		PaymentTransactionAmount:     req.Amount,
		PaymentTransactionReceiverID: account.AccountID,
		PaymentTransactionCreatedAt:  now,
		PaymentTransactionUpdatedAt:  now,
	}); err != nil {
		log.Errorw("Failed to apply deposit event", "error", err)
		return nil, stackerr.Error(err)
	}

	publisher := eventpkg.NewPublisher(h.outboxRepo)
	if err := publisher.PublishAggregate(ctx, agg); err != nil {
		log.Errorw("Failed to publish deposit event", "error", err)
		return nil, stackerr.Error(err)
	}

	return &out.DepositResponse{
		Message: "Deposit successful",
	}, nil
}
