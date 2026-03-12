package command

import (
	"go-socket/core/modules/payment/application/dto/in"
	"go-socket/core/modules/payment/application/dto/out"
	"go-socket/core/shared/pkg/cqrs"
)

type DepositHandler = cqrs.Handler[*in.DepositRequest, *out.DepositResponse]
type TransferHandler = cqrs.Handler[*in.TransferRequest, *out.TransferResponse]
type WithdrawalHandler = cqrs.Handler[*in.WithdrawalRequest, *out.WithdrawalResponse]

type Bus struct {
	Deposit    cqrs.Dispatcher[*in.DepositRequest, *out.DepositResponse]
	Transfer   cqrs.Dispatcher[*in.TransferRequest, *out.TransferResponse]
	Withdrawal cqrs.Dispatcher[*in.WithdrawalRequest, *out.WithdrawalResponse]
}

func NewBus(depositHandler DepositHandler, transferHandler TransferHandler, withdrawalHandler WithdrawalHandler) Bus {
	return Bus{
		Deposit:    cqrs.NewDispatcher(depositHandler),
		Transfer:   cqrs.NewDispatcher(transferHandler),
		Withdrawal: cqrs.NewDispatcher(withdrawalHandler),
	}
}
