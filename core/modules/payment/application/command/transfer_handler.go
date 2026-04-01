package command

import (
	"context"
	"errors"
	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/domain/repos"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type transferHandler struct {
	baseRepo                     repos.Repos
	paymentAccountProjectionRepo repos.PaymentAccountProjectionRepository
}

func NewTransferHandler(repos repos.Repos) TransferHandler {
	return &transferHandler{
		baseRepo: repos,

		paymentAccountProjectionRepo: repos.PaymentAccountProjectionRepository(),
	}
}

func (h *transferHandler) Handle(ctx context.Context, req *in.TransferRequest) (*out.TransferResponse, error) {
	log := logging.FromContext(ctx).Named("Transfer")

	accountID, err := accountIDFromContext(ctx)
	if err != nil {
		log.Errorw("Account not found")
		return nil, stackerr.Error(err)
	}
	receiverID := req.ReceiverID
	receiver, err := h.paymentAccountProjectionRepo.GetAccountProjectionByAccountID(ctx, receiverID)
	if err != nil {
		log.Errorw("Failed to get receiver account projection", zap.Error(err))
		return nil, stackerr.Error(err)
	}
	if receiver == nil {
		log.Errorw("Receiver account projection not found")
		return nil, stackerr.Error(errors.New("receiver account projection not found"))
	}

	transactionID := uuid.NewString()
	now := time.Now().UTC()
	var (
		balance int64
		version int
	)

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		senderAgg, err := txRepos.PaymentBalanceAggregateRepository().Load(ctx, accountID)
		if err != nil {
			return stackerr.Error(err)
		}
		if err := senderAgg.Transfer(transactionID, req.Amount, receiverID, now); err != nil {
			return stackerr.Error(err)
		}
		if err := txRepos.PaymentBalanceAggregateRepository().Save(ctx, senderAgg); err != nil {
			return stackerr.Error(err)
		}

		receiverAgg, err := txRepos.PaymentBalanceAggregateRepository().Load(ctx, receiverID)
		if err != nil {
			return stackerr.Error(err)
		}
		if err := receiverAgg.Receive(transactionID, req.Amount, accountID, now); err != nil {
			return stackerr.Error(err)
		}
		if err := txRepos.PaymentBalanceAggregateRepository().Save(ctx, receiverAgg); err != nil {
			return stackerr.Error(err)
		}

		balance = senderAgg.Balance
		version = senderAgg.Root().Version()

		return nil
	}); err != nil {
		log.Errorw("Handler transfer failed", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	return &out.TransferResponse{
		Message:       "Transfer successful",
		TransactionID: transactionID,
		Balance:       balance,
		Version:       version,
	}, nil
}
