package command

import (
	"context"
	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/modules/payment/domain/repos"
)

type transferHandler struct {
}

func NewTransferHandler(repos repos.Repos) TransferHandler {
	return &transferHandler{}
}

func (h *transferHandler) Handle(ctx context.Context, req *in.TransferRequest) (*out.TransferResponse, error) {
	return &out.TransferResponse{
		Message: "Transfer successful",
	}, nil
}
