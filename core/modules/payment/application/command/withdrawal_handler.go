package command

import (
	"context"
	"go-socket/core/modules/payment/domain/repos"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"
	"time"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"

	"github.com/google/uuid"
	"go-socket/core/shared/pkg/cqrs"
	"go.uber.org/zap"
)

type withdrawalHandler struct {
	baseRepo repos.Repos
}

func NewWithdrawalHandler(repos repos.Repos) cqrs.Handler[*in.WithdrawalRequest, *out.WithdrawalResponse] {
	return &withdrawalHandler{
		baseRepo: repos,
	}
}

func (h *withdrawalHandler) Handle(ctx context.Context, req *in.WithdrawalRequest) (*out.WithdrawalResponse, error) {
	log := logging.FromContext(ctx).Named("Withdrawal")

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

		if err := agg.Withdraw(transactionID, req.Amount, now); err != nil {
			return stackerr.Error(err)
		}

		if err := txRepos.PaymentBalanceAggregateRepository().Save(ctx, agg); err != nil {
			return stackerr.Error(err)
		}

		balance = agg.Balance
		version = agg.Root().Version()
		return nil
	}); err != nil {
		log.Errorw("Failed to withdraw", zap.Error(err))
		return nil, stackerr.Error(err)
	}

	return &out.WithdrawalResponse{
		Message:       "Withdrawal successful",
		TransactionID: transactionID,
		Balance:       balance,
		Version:       version,
	}, nil
}
