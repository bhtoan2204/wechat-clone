package command

import (
	"context"
	"time"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type depositHandler struct {
	baseRepo repos.Repos
}

func NewDepositHandler(repos repos.Repos) cqrs.Handler[*in.DepositRequest, *out.DepositResponse] {
	return &depositHandler{
		baseRepo: repos,
	}
}

func (h *depositHandler) Handle(ctx context.Context, req *in.DepositRequest) (*out.DepositResponse, error) {
	log := logging.FromContext(ctx).Named("Deposit")

	accountID, err := accountIDFromContext(ctx)
	if err != nil {
		log.Errorw("Account not found")
		return nil, stackerr.Error(err)
	}

	now := time.Now().UTC()
	transactionID := uuid.NewString()
	var (
		balance int64
		version int
	)

	if err := h.baseRepo.WithTransaction(ctx, func(txRepos repos.Repos) error {
		agg, err := txRepos.PaymentBalanceAggregateRepository().Load(ctx, accountID)
		if err != nil {
			return stackerr.Error(err)
		}

		if err := agg.Deposit(transactionID, req.Amount, now); err != nil {
			return stackerr.Error(err)
		}

		if err := txRepos.PaymentBalanceAggregateRepository().Save(ctx, agg); err != nil {
			return stackerr.Error(err)
		}

		balance = agg.Balance
		version = agg.Root().Version()
		return nil
	}); err != nil {
		log.Errorw("Failed to deposit", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	return &out.DepositResponse{
		Message:       "Deposit successful",
		TransactionID: transactionID,
		Balance:       balance,
		Version:       version,
	}, nil
}
