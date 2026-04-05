package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"go-socket/scaffold/models"
	"go-socket/scaffold/utils"
)

func GenerateHandler(endpoints []models.Endpoint) (string, error) {
	tmpl, err := template.ParseFiles("scaffold/template/handler.tmpl")
	if err != nil {
		return "", err
	}
	if len(endpoints) == 0 {
		return "", errors.New("no endpoints to generate handler")
	}

	seen := make(map[string]bool)
	created := 0
	skipped := 0
	for _, ep := range endpoints {
		if !shouldGenerateHandler(ep) {
			continue
		}
		module, err := moduleForUsecase(ep.Usecase.Name)
		if err != nil {
			return "", err
		}
		key := module.FsRoot + ":handler:" + ep.Handler
		if seen[key] {
			continue
		}
		seen[key] = true

		written, err := writeHandlerFile(tmpl, module, ep)
		if err != nil {
			return "", err
		}
		if written {
			created++
		} else {
			skipped++
		}
	}

	return fmt.Sprintf("generated %d handler(s), skipped %d existing file(s)", created, skipped), nil
}

func shouldGenerateHandler(ep models.Endpoint) bool {
	if ep.Handler == "" || ep.Usecase.Method == "" || ep.Usecase.Name == "" {
		return false
	}
	if ep.Request.Struct == "" || ep.Response.Struct == "" {
		return false
	}
	return true
}

func writeHandlerFile(tmpl *template.Template, module modulePaths, ep models.Endpoint) (bool, error) {
	dispatcherField := lowerFirst(ep.Usecase.Method)

	data := handlerTemplateData{
		PackageName:       "handler",
		HandlerName:       ep.Handler,
		StructName:        lowerFirst(strings.TrimSuffix(ep.Handler, "Handler")) + "Handler",
		Method:            strings.ToUpper(ep.Method),
		RequestStruct:     ep.Request.Struct,
		ResponseStruct:    ep.Response.Struct,
		RequestDtoImport:  module.ImportRoot + "/application/dto/in",
		ResponseDtoImport: module.ImportRoot + "/application/dto/out",
		DispatcherImport:  "go-socket/core/shared/pkg/cqrs",
		DispatcherPackage: "cqrs",
		DispatcherField:   dispatcherField,
		RequestInit:       buildRequestInit(ep),
		ActionName:        ep.Usecase.Method,
	}

	fileName := utils.Snake(ep.Handler) + "_handler.go"
	dst := filepath.Join(module.FsRoot, "transport/http/handler", fileName)

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, err
	}
	if fileExists(dst) {
		return false, nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return false, err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return false, fmt.Errorf("format handler failed: %w", err)
	}

	if err := os.WriteFile(dst, formatted, 0o644); err != nil {
		return false, err
	}
	return true, nil
}

type handlerTemplateData struct {
	PackageName       string
	HandlerName       string
	StructName        string
	Method            string
	RequestStruct     string
	ResponseStruct    string
	RequestDtoImport  string
	ResponseDtoImport string
	DispatcherImport  string
	DispatcherPackage string
	DispatcherField   string
	RequestInit       string
	ActionName        string
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func buildRequestInit(ep models.Endpoint) string {
	paramPattern := regexp.MustCompile(`:([A-Za-z0-9_]+)`)
	matches := paramPattern.FindAllStringSubmatch(ep.Path, -1)
	if len(matches) == 0 {
		return ""
	}

	assignments := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		paramName := match[1]
		fieldName := utils.Pascal(paramName)
		if !requestHasField(ep.Request.Fields, paramName) {
			continue
		}
		assignments = append(assignments, fieldName+`: c.Param("`+paramName+`")`)
	}
	if len(assignments) == 0 {
		return ""
	}

	return "request := in." + ep.Request.Struct + "{" + strings.Join(assignments, ", ") + "}"
}

func requestHasField(fields []models.FieldSpec, name string) bool {
	for _, field := range fields {
		if field.Name == name {
			return true
		}
	}
	return false
}
