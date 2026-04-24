package cronjob

import (
	"time"

	paymenttask "wechat-clone/core/modules/payment/application/scheduler/task"
	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/hibiken/asynq"
)

type CronJob interface {
	Start() error
	Stop() error
}

type cronJob struct {
	scheduler *asynq.Scheduler
}

func NewCronJob(scheduler *asynq.Scheduler, interval time.Duration) (CronJob, error) {
	if scheduler == nil {
		return &cronJob{}, nil
	}

	task := asynq.NewTask(paymenttask.ProcessPendingWithdrawalsTask, nil)
	if _, err := scheduler.Register(
		paymenttask.PeriodicSpec(interval),
		task,
		asynq.Queue(paymenttask.QueueName),
		asynq.MaxRetry(0),
		asynq.Unique(interval),
	); err != nil {
		return nil, stackErr.Error(err)
	}

	return &cronJob{scheduler: scheduler}, nil
}

func (j *cronJob) Start() error {
	if j == nil || j.scheduler == nil {
		return nil
	}

	if err := j.scheduler.Start(); err != nil {
		return stackErr.Error(err)
	}

	return nil
}

func (j *cronJob) Stop() error {
	if j == nil || j.scheduler == nil {
		return nil
	}

	j.scheduler.Shutdown()
	return nil
}
