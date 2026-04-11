package messaging

import (
	"fmt"

	"go-socket/core/shared/pkg/stackErr"
)

type Config struct {
	Servers      string
	Group        string
	OffsetReset  string
	ConsumeTopic []string
	HandlerName  string
	DLQ          bool
}

func (c *Config) Validate() error {
	if len(c.Servers) == 0 {
		return stackErr.Error(fmt.Errorf("server cant empty"))
	}

	if len(c.ConsumeTopic) == 0 {
		return stackErr.Error(fmt.Errorf("do not have any topic"))
	}

	return nil
}
