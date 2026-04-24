package assembly

import (
	appCtx "wechat-clone/core/context"
	paymenttask "wechat-clone/core/modules/payment/application/scheduler/task"
	"wechat-clone/core/modules/payment/application/scheduler/taskhandler"
	paymentservice "wechat-clone/core/modules/payment/application/service"
	paymentrepo "wechat-clone/core/modules/payment/infra/persistent/repository"
	provideradapter "wechat-clone/core/modules/payment/infra/provider"
	"wechat-clone/core/modules/payment/providers"
	mockprovider "wechat-clone/core/modules/payment/providers/mock"
	stripeprovider "wechat-clone/core/modules/payment/providers/stripe"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
	modruntime "wechat-clone/core/shared/runtime"

	"github.com/hibiken/asynq"
)

func buildTaskRuntime(_ *config.Config, appContext *appCtx.AppContext) (modruntime.Module, error) {
	commandService, err := buildPaymentCommandService(appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	server, err := newAsynqServer(appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return taskhandler.NewTaskHandler(commandService, server), nil
}

func buildPaymentCommandService(appContext *appCtx.AppContext) (paymentservice.PaymentCommandService, error) {
	paymentRepos := paymentrepo.NewRepoImpl(appContext)
	providerRegistry := providers.NewProviderRegistry()
	providerRegistry.Register(mockprovider.NewProvider(appContext.GetConfig().LedgerConfig.MockWebhookSecret))
	if stripe := stripeprovider.NewProvider(appContext.GetConfig().LedgerConfig.Stripe); stripe.Enabled() {
		providerRegistry.Register(stripe)
	}

	return paymentservice.NewPaymentCommandService(
		appContext,
		paymentRepos,
		provideradapter.NewPaymentProviderRegistry(providerRegistry),
	), nil
}

func newAsynqServer(appContext *appCtx.AppContext) (*asynq.Server, error) {
	redisConnOpt, err := newAsynqRedisConnOpt(appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return asynq.NewServer(redisConnOpt, asynq.Config{
		Concurrency: 1,
		Queues: map[string]int{
			paymenttask.QueueName: 1,
		},
	}), nil
}
