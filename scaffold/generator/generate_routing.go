package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"go-socket/scaffold/models"
)

func GenerateRouting(spec *models.APISpec) (string, error) {
	if spec == nil {
		return "", errors.New("api spec is nil")
	}
	if len(spec.Endpoints) == 0 {
		return "", errors.New("no endpoints to generate routing")
	}

	tmpl, err := template.ParseFiles("scaffold/template/routing.tmpl")
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
		dst := filepath.Join(group.Module.FsRoot, "transport/http/routes.go")
		if fileExists(dst) {
			skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}

		data := buildRoutingTemplateData(group)
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}

		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			return "", fmt.Errorf("format routing failed: %v", err)
		}

		if err := os.WriteFile(dst, formatted, 0o644); err != nil {
			return "", err
		}
		created++
	}

	return fmt.Sprintf("generated %d routing file(s), skipped %d existing file(s)", created, skipped), nil
}

type routingTemplateData struct {
	PackageName       string
	RequestDtoImport  string
	ResponseDtoImport string
	HandlerImport     string
	DispatcherImport  string
	WrapperImport     string
	PublicParams      []routingParam
	PrivateParams     []routingParam
	PublicRoutes      []routingRoute
	PrivateRoutes     []routingRoute
}

type routingParam struct {
	Name         string
	RequestType  string
	ResponseType string
}

type routingRoute struct {
	Method      string
	Path        string
	HandlerName string
	ParamName   string
}

func buildRoutingTemplateData(group moduleEndpoints) routingTemplateData {
	data := routingTemplateData{
		PackageName:       "http",
		RequestDtoImport:  group.Module.ImportRoot + "/application/dto/in",
		ResponseDtoImport: group.Module.ImportRoot + "/application/dto/out",
		HandlerImport:     group.Module.ImportRoot + "/transport/http/handler",
		DispatcherImport:  "go-socket/core/shared/pkg/cqrs",
		WrapperImport:     "go-socket/core/shared/transport/httpx",
	}

	publicSeen := make(map[string]bool)
	privateSeen := make(map[string]bool)

	for _, ep := range group.Endpoints {
		param := routingParam{
			Name:         dispatcherParamName(ep),
			RequestType:  requestType(ep.Request),
			ResponseType: responseType(ep.Response),
		}
		route := routingRoute{
			Method:      stringsToUpper(ep.Method),
			Path:        ep.Path,
			HandlerName: ep.Handler,
			ParamName:   param.Name,
		}

		if ep.Auth {
			if !privateSeen[param.Name] {
				data.PrivateParams = append(data.PrivateParams, param)
				privateSeen[param.Name] = true
			}
			data.PrivateRoutes = append(data.PrivateRoutes, route)
			continue
		}

		if !publicSeen[param.Name] {
			data.PublicParams = append(data.PublicParams, param)
			publicSeen[param.Name] = true
		}
		data.PublicRoutes = append(data.PublicRoutes, route)
	}

	return data
}

func stringsToUpper(s string) string {
	return strings.ToUpper(s)
}
