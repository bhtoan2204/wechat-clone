package modruntime

import (
	"fmt"

	"go-socket/core/shared/pkg/stackErr"
)

type compositeModule struct {
	modules []Module
}

func NewComposite(modules ...Module) Module {
	filtered := make([]Module, 0, len(modules))
	for _, module := range modules {
		if module != nil {
			filtered = append(filtered, module)
		}
	}
	return &compositeModule{modules: filtered}
}

func (m *compositeModule) Start() error {
	for idx, module := range m.modules {
		if err := module.Start(); err != nil {
			m.stopStarted(idx - 1)
			return stackErr.Error(fmt.Errorf("start runtime %T failed: %v", module, err))
		}
	}
	return nil
}

func (m *compositeModule) Stop() error {
	var firstErr error
	for idx := len(m.modules) - 1; idx >= 0; idx-- {
		if err := m.modules[idx].Stop(); err != nil && firstErr == nil {
			firstErr = stackErr.Error(fmt.Errorf("stop runtime %T failed: %v", m.modules[idx], err))
		}
	}
	return firstErr
}

func (m *compositeModule) stopStarted(lastIdx int) {
	for idx := lastIdx; idx >= 0; idx-- {
		_ = m.modules[idx].Stop()
	}
}
