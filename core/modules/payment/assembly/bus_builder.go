package assembly

import (
	appCtx "go-socket/core/context"
	"go-socket/core/modules/payment/application/command"
	paymentrepo "go-socket/core/modules/payment/infra/persistent/repository"
)

func BuildBuses(appCtx *appCtx.AppContext) command.Bus {
	paymentRepos := paymentrepo.NewRepoImpl(appCtx)
	depositHandler := command.NewDepositHandler(paymentRepos)
	transferHandler := command.NewTransferHandler(paymentRepos)
	withdrawalHandler := command.NewWithdrawalHandler(paymentRepos)
	return command.NewBus(depositHandler, transferHandler, withdrawalHandler)
}
