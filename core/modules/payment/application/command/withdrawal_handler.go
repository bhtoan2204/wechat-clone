package command

import (
	"context"
	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/domain/repos"
)

type withdrawalHandler struct {
}

func NewWithdrawalHandler(repos repos.Repos) WithdrawalHandler {
	return &withdrawalHandler{}
}

func (h *withdrawalHandler) Handle(ctx context.Context, req *in.WithdrawalRequest) (*out.WithdrawalResponse, error) {
	return &out.WithdrawalResponse{
		Message: "Withdrawal successful",
	}, nil
}
