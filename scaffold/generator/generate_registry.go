package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"go-socket/scaffold/models"
)

func GenerateRegistry(spec *models.APISpec) (string, error) {
	if spec == nil {
		return "", errors.New("api spec is nil")
	}
	if len(spec.Endpoints) == 0 {
		return "", errors.New("no endpoints to generate registry")
	}

	tmpl, err := template.ParseFiles("scaffold/template/registry.tmpl")
	if err != nil {
		return "", err
	}

	groups, err := groupEndpointsByModule(spec.Endpoints)
	if err != nil {
		return "", err
	}

	created := 0
	skipped := 0
	for _, group := range groups {
		dst := serverTargetPath(group.Module)
		if fileExists(dst) {
			skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}

		data := buildRegistryTemplateData(group)
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			return "", fmt.Errorf("format registry failed: %v", err)
		}

		if err := os.WriteFile(dst, formatted, 0o644); err != nil {
			return "", err
		}
		created++
	}

	return fmt.Sprintf("generated %d server file(s), skipped %d existing file(s)", created, skipped), nil
}

type registryTemplateData struct {
	PackageName       string
	ServerStructName  string
	ModuleHTTPAlias   string
	RequestDtoImport  string
	ResponseDtoImport string
	ModuleHTTPImport  string
	DispatcherImport  string
	HTTPImport        string
	Fields            []registryParam
	PublicParams      []registryParam
	PrivateParams     []registryParam
}

type registryParam struct {
	Name         string
	RequestType  string
	ResponseType string
}

func buildRegistryTemplateData(group moduleEndpoints) registryTemplateData {
	moduleName := modulePackageName(group.Module.ImportRoot)
	data := registryTemplateData{
		PackageName:       "server",
		ServerStructName:  lowerFirst(moduleName) + "HTTPServer",
		ModuleHTTPAlias:   moduleName + "http",
		RequestDtoImport:  group.Module.ImportRoot + "/application/dto/in",
		ResponseDtoImport: group.Module.ImportRoot + "/application/dto/out",
		ModuleHTTPImport:  group.Module.ImportRoot + "/transport/http",
		DispatcherImport:  "go-socket/core/shared/pkg/cqrs",
		HTTPImport:        "go-socket/core/shared/transport/http",
	}

	fieldSeen := make(map[string]bool)
	publicSeen := make(map[string]bool)
	privateSeen := make(map[string]bool)

	for _, ep := range group.Endpoints {
		param := registryParam{
			Name:         dispatcherParamName(ep),
			RequestType:  requestType(ep.Request),
			ResponseType: responseType(ep.Response),
		}

		if !fieldSeen[param.Name] {
			data.Fields = append(data.Fields, param)
			fieldSeen[param.Name] = true
		}

		if ep.Auth {
			if !privateSeen[param.Name] {
				data.PrivateParams = append(data.PrivateParams, param)
				privateSeen[param.Name] = true
			}
			continue
		}

		if !publicSeen[param.Name] {
			data.PublicParams = append(data.PublicParams, param)
			publicSeen[param.Name] = true
		}
	}

	return data
}

func serverTargetPath(module modulePaths) string {
	serverDir := filepath.Join(module.FsRoot, "transport/server")
	serverFile := filepath.Join(serverDir, "server.go")
	if fileExists(serverFile) {
		return serverFile
	}

	httpServerFile := filepath.Join(serverDir, "http_server.go")
	if fileExists(httpServerFile) {
		return httpServerFile
	}

	return httpServerFile
}
