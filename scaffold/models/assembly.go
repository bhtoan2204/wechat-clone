package models

import (
	"fmt"
	"os"

	"wechat-clone/core/shared/pkg/stackErr"

	"github.com/goccy/go-yaml"
)

type AssemblySpec struct {
	Modules []AssemblyModule `json:"modules" yaml:"modules"`
}

type AssemblyModule struct {
	Name  string         `json:"name" yaml:"name"`
	Kinds []AssemblyKind `json:"kinds" yaml:"kinds"`
}

type AssemblyKind string

const (
	AssemblyKindHTTP       AssemblyKind = "http"
	AssemblyKindMessaging  AssemblyKind = "messaging"
	AssemblyKindProjection AssemblyKind = "projection"
	AssemblyKindTask       AssemblyKind = "task"
	AssemblyKindCron       AssemblyKind = "cron"
)

func LoadAssemblySpec(path string) (*AssemblySpec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, stackErr.Error(err)
	}

	var spec AssemblySpec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, stackErr.Error(err)
	}
	if err := spec.Validate(); err != nil {
		return nil, stackErr.Error(err)
	}
	return &spec, nil
}

func (s *AssemblySpec) Validate() error {
	if s == nil {
		return fmt.Errorf("assembly spec is nil")
	}
	if len(s.Modules) == 0 {
		return fmt.Errorf("assembly spec has no modules")
	}

	for _, module := range s.Modules {
		if module.Name == "" {
			return fmt.Errorf("assembly module name is required")
		}
		if len(module.Kinds) == 0 {
			return fmt.Errorf("assembly module %s has no kinds", module.Name)
		}
		for _, kind := range module.Kinds {
			switch kind {
			case AssemblyKindHTTP, AssemblyKindMessaging, AssemblyKindProjection, AssemblyKindTask, AssemblyKindCron:
			default:
				return fmt.Errorf("assembly module %s has unsupported kind %s", module.Name, kind)
			}
		}
	}

	return nil
}
