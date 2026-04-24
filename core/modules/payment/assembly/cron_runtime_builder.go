package assembly

import (
	"time"

	appCtx "wechat-clone/core/context"
	"wechat-clone/core/modules/payment/application/scheduler/cronjob"
	"wechat-clone/core/shared/config"
	"wechat-clone/core/shared/pkg/stackErr"
	modruntime "wechat-clone/core/shared/runtime"

	"github.com/hibiken/asynq"
)

func buildCronRuntime(cfg *config.Config, appContext *appCtx.AppContext) (modruntime.Module, error) {
	scheduler, err := newAsynqScheduler(appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	interval := time.Duration(cfg.LedgerConfig.Stripe.WithdrawalScheduleIntervalSecond) * time.Second
	job, err := cronjob.NewCronJob(scheduler, interval)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return job, nil
}

func newAsynqScheduler(appContext *appCtx.AppContext) (*asynq.Scheduler, error) {
	redisConnOpt, err := newAsynqRedisConnOpt(appContext)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	return asynq.NewScheduler(redisConnOpt, &asynq.SchedulerOpts{}), nil
}

func newAsynqRedisConnOpt(appContext *appCtx.AppContext) (asynq.RedisClientOpt, error) {
	if appContext == nil || appContext.GetRedisClient() == nil {
		return asynq.RedisClientOpt{}, nil
	}

	redisOptions := appContext.GetRedisClient().Options()
	if redisOptions == nil {
		return asynq.RedisClientOpt{}, nil
	}

	return asynq.RedisClientOpt{
		Addr:     redisOptions.Addr,
		Username: redisOptions.Username,
		Password: redisOptions.Password,
		DB:       redisOptions.DB,
	}, nil
}
