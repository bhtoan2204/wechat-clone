package models

import (
	"fmt"
	"go-socket/core/shared/pkg/stackErr"
	"os"
	"path/filepath"
	"sort"

	"github.com/goccy/go-yaml"
)

type APISpec struct {
	Version   int        `json:"version" yaml:"version"`
	BasePath  string     `json:"basePath" yaml:"basePath"`
	Endpoints []Endpoint `json:"endpoints" yaml:"endpoints"`
}

type Endpoint struct {
	Name          string  `json:"name" yaml:"name"`
	Method        string  `json:"method" yaml:"method"`
	Path          string  `json:"path" yaml:"path"`
	Handler       string  `json:"handler" yaml:"handler"`
	Auth          bool    `json:"auth,omitempty" yaml:"auth,omitempty"`
	SuccessStatus int     `json:"successStatus,omitempty" yaml:"successStatus,omitempty"`
	Usecase       Usecase `json:"usecase" yaml:"usecase"`
	Request       Payload `json:"request" yaml:"request"`
	Response      Payload `json:"response" yaml:"response"`
}

type Usecase struct {
	Name   string `json:"name" yaml:"name"`
	Method string `json:"method" yaml:"method"`
}

type Payload struct {
	Struct     string      `json:"struct" yaml:"struct"`
	Collection bool        `json:"collection,omitempty" yaml:"collection,omitempty"`
	Fields     []FieldSpec `json:"fields" yaml:"fields"`
}

type FieldSpec struct {
	Name     string   `json:"name" yaml:"name"`
	Type     string   `json:"type" yaml:"type"`
	Struct   string   `json:"struct,omitempty" yaml:"struct,omitempty"`
	Source   string   `json:"source,omitempty" yaml:"source,omitempty"`
	Header   string   `json:"header,omitempty" yaml:"header,omitempty"`
	Items    *Payload `json:"items,omitempty" yaml:"items,omitempty"`
	Required bool     `json:"required,omitempty" yaml:"required,omitempty"`
	Pointer  bool     `json:"pointer,omitempty" yaml:"pointer,omitempty"`
}

func LoadAPISpec(path string) (*APISpec, error) {
	yamlFile, err := os.ReadFile(path)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	var apiSpec APISpec
	err = yaml.Unmarshal(yamlFile, &apiSpec)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	return &apiSpec, nil
}

func LoadAPISpecDir(dir string) (*APISpec, error) {
	pattern := filepath.Join(dir, "*.yaml")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return nil, stackErr.Error(err)
	}
	if len(files) == 0 {
		return nil, stackErr.Error(fmt.Errorf("no api spec files found in %s", dir))
	}
	sort.Strings(files)
	merged := &APISpec{}
	for _, file := range files {
		spec, err := LoadAPISpec(file)
		if err != nil {
			return nil, stackErr.Error(err)
		}
		if merged.Version == 0 {
			merged.Version = spec.Version
		}
		if merged.BasePath == "" {
			merged.BasePath = spec.BasePath
		}
		if spec.BasePath != "" && merged.BasePath != spec.BasePath {
			return nil, stackErr.Error(fmt.Errorf("basePath mismatch in %s", file))
		}
		merged.Endpoints = append(merged.Endpoints, spec.Endpoints...)
	}
	return merged, nil
}
