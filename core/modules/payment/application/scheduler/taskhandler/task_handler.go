package taskhandler

import (
	"context"

	paymenttask "wechat-clone/core/modules/payment/application/scheduler/task"
	paymentservice "wechat-clone/core/modules/payment/application/service"
	"wechat-clone/core/shared/pkg/logging"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/hibiken/asynq"
	"go.uber.org/zap"
)

type TaskHandler interface {
	Start() error
	Stop() error
}

type taskHandler struct {
	service paymentservice.PaymentCommandService
	server  *asynq.Server
}

func NewTaskHandler(service paymentservice.PaymentCommandService, server *asynq.Server) TaskHandler {
	if service == nil || server == nil {
		return &taskHandler{}
	}
	return &taskHandler{
		service: service,
		server:  server,
	}
}

func (h *taskHandler) Start() error {
	if h == nil || h.service == nil || h.server == nil {
		return nil
	}

	mux := asynq.NewServeMux()
	mux.HandleFunc(paymenttask.ProcessPendingWithdrawalsTask, h.handleProcessPendingWithdrawals)

	if err := h.server.Start(mux); err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (h *taskHandler) Stop() error {
	if h == nil || h.server == nil {
		return nil
	}

	h.server.Shutdown()
	return nil
}

func (h *taskHandler) handleProcessPendingWithdrawals(ctx context.Context, _ *asynq.Task) error {
	if h == nil || h.service == nil {
		return nil
	}

	if err := h.service.ProcessPendingWithdrawals(ctx); err != nil {
		logging.FromContext(ctx).Warnw("process pending withdrawals failed", zap.Error(err))
		return stackErr.Error(err)
	}

	return nil
}
