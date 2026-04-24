package task

import (
	"fmt"
	"time"
)

const (
	ProcessPendingWithdrawalsTask = "payment:withdrawal:process-pending"
	QueueName                     = "payment:scheduler"
)

func PeriodicSpec(interval time.Duration) string {
	seconds := int(interval / time.Second)
	if seconds <= 0 {
		seconds = 5
	}
	return fmt.Sprintf("@every %ds", seconds)
}
