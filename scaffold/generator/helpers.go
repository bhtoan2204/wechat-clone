package generator

import (
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"go-socket/core/shared/pkg/stackErr"
	"go-socket/scaffold/models"
)

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func structExistsInDir(dir, structName string) bool {
	return structExistsInDirExcept(dir, structName, "")
}

func structExistsInDirExcept(dir, structName, excludePath string) bool {
	found := false
	excludePath = filepath.Clean(excludePath)
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || found || d == nil || d.IsDir() || filepath.Ext(path) != ".go" {
			return nil
		}
		if excludePath != "" && filepath.Clean(path) == excludePath {
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		if strings.Contains(string(content), "type "+structName+" struct") {
			found = true
		}
		return nil
	})
	return found
}

func isGeneratedFile(path, kind string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	marker := "// CODE_GENERATOR:"
	doNotEditMarker := "// CODE_GENERATOR - do not edit:"
	if kind != "" {
		marker = "// CODE_GENERATOR: " + kind
		doNotEditMarker = "// CODE_GENERATOR - do not edit: " + kind
	}
	text := string(content)
	return strings.Contains(text, marker) || strings.Contains(text, doNotEditMarker)
}

func isScaffoldStubFile(path string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(content), `fmt.Errorf("not implemented yet")`)
}

type moduleEndpoints struct {
	Module    modulePaths
	Endpoints []models.Endpoint
}

func groupEndpointsByModule(endpoints []models.Endpoint) ([]moduleEndpoints, error) {
	groups := make([]moduleEndpoints, 0)
	indexByRoot := make(map[string]int)

	for _, ep := range endpoints {
		module, err := moduleForUsecase(ep.Usecase.Name)
		if err != nil {
			return nil, stackErr.Error(err)
		}

		idx, ok := indexByRoot[module.FsRoot]
		if !ok {
			idx = len(groups)
			indexByRoot[module.FsRoot] = idx
			groups = append(groups, moduleEndpoints{Module: module})
		}
		groups[idx].Endpoints = append(groups[idx].Endpoints, ep)
	}

	return groups, nil
}

func modulePackageName(importRoot string) string {
	return path.Base(importRoot)
}

func dispatcherParamName(ep models.Endpoint) string {
	return lowerFirst(ep.Usecase.Method)
}

func requestType(req models.Payload) string {
	if req.Struct == "" {
		return "interface{}"
	}
	return "*in." + req.Struct
}

func responseType(resp models.Payload) string {
	if resp.Struct == "" {
		return "interface{}"
	}
	if resp.Collection {
		return "[]*out." + resp.Struct
	}
	return "*out." + resp.Struct
}
