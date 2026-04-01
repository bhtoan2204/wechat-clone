package command

import (
	"context"

	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	paymentrepos "go-socket/core/modules/payment/domain/repos"
	"go-socket/core/shared/pkg/cqrs"
	"go-socket/core/shared/pkg/logging"
	stackerr "go-socket/core/shared/pkg/stackErr"

	"go.uber.org/zap"
)

type rebuildProjectionHandler struct {
	baseRepo paymentrepos.Repos
}

func NewRebuildProjectionHandler(repos paymentrepos.Repos) cqrs.Handler[*in.RebuildProjectionRequest, *out.RebuildProjectionResponse] {
	return &rebuildProjectionHandler{
		baseRepo: repos,
	}
}

func (h *rebuildProjectionHandler) Handle(ctx context.Context, req *in.RebuildProjectionRequest) (*out.RebuildProjectionResponse, error) {
	log := logging.FromContext(ctx).Named("RebuildPaymentProjection")

	var result *paymentrepos.ProjectionRebuildResult
	mode := paymentrepos.ProjectionRebuildMode(req.Mode)
	if err := h.baseRepo.WithTransaction(ctx, func(txRepos paymentrepos.Repos) error {
		rebuildResult, err := txRepos.PaymentProjectionRepository().RebuildProjection(ctx, req.AccountID, mode)
		if err != nil {
			return err
		}
		result = rebuildResult
		return nil
	}); err != nil {
		log.Errorw("failed to rebuild payment projection", zap.Error(err), zap.String("mode", req.Mode), zap.String("account_id", req.AccountID))
		return nil, stackerr.Error(err)
	}

	response := &out.RebuildProjectionResponse{
		Message:             "Projection rebuild completed",
		Mode:                req.Mode,
		AccountID:           req.AccountID,
		Accounts:            result.Accounts,
		EventsReplayed:      result.EventsReplayed,
		TransactionsRebuilt: result.TransactionsRebuilt,
		BalancesRebuilt:     result.BalancesRebuilt,
	}
	if mode == paymentrepos.ProjectionRebuildModeSnapshot {
		response.Note = "Snapshot mode rebuilds payment_balances from the latest snapshot and newer events; historical payment_transactions are left unchanged."
	}

	return response, nil
}
