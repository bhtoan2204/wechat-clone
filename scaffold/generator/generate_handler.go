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
			// continue
		}
		module, err := moduleForUsecase(ep.Usecase.Name)
		if err != nil {
			return "", err
		}
		key := module.FsRoot + ":handler:" + ep.Handler
		if seen[key] {
			// continue
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
		ResponseType:      responseType(ep.Response),
		RequestDtoImport:  module.ImportRoot + "/application/dto/in",
		ResponseDtoImport: module.ImportRoot + "/application/dto/out",
		DispatcherImport:  "go-socket/core/shared/pkg/cqrs",
		DispatcherPackage: "cqrs",
		DispatcherField:   dispatcherField,
		RequestSetup:      buildRequestSetup(ep),
		NeedsIO:           endpointNeedsRawBody(ep),
		BindMode:          handlerBindMode(ep),
		SuccessStatus:     ep.SuccessStatus,
		ActionName:        ep.Usecase.Method,
	}

	fileName := utils.Snake(ep.Handler) + "_handler.go"
	dst := filepath.Join(module.FsRoot, "transport/http/handler", fileName)

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return false, err
	}
	if fileExists(dst) && !isGeneratedFile(dst, "handler") {
		return false, nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return false, err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return false, fmt.Errorf("format handler failed: %v", err)
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
	ResponseType      string
	RequestDtoImport  string
	ResponseDtoImport string
	DispatcherImport  string
	DispatcherPackage string
	DispatcherField   string
	RequestSetup      string
	NeedsIO           bool
	BindMode          string
	SuccessStatus     int
	ActionName        string
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func buildRequestSetup(ep models.Endpoint) string {
	lines := []string{"var request in." + ep.Request.Struct}
	hasSetup := false

	for _, field := range ep.Request.Fields {
		source := fieldSource(ep, field)
		fieldName := utils.Pascal(field.Name)

		switch source {
		case "path":
			lines = append(lines, `request.`+fieldName+` = c.Param("`+field.Name+`")`)
			hasSetup = true
		case "header":
			headerName := field.Header
			if headerName == "" {
				headerName = field.Name
			}
			lines = append(lines, `request.`+fieldName+` = c.GetHeader("`+headerName+`")`)
			hasSetup = true
		case "raw_body":
			lines = append(lines,
				"",
				"payload, err := io.ReadAll(c.Request.Body)",
				"if err != nil {",
				`	logger.Errorw("Read request body failed", zap.Error(err))`,
				`	c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "unable to read request body"})`,
				"	return nil, nil",
				"}",
				`request.`+fieldName+` = string(payload)`,
			)
			hasSetup = true
		}
	}

	if !hasSetup {
		return ""
	}

	return strings.Join(lines, "\n")
}

func requestHasField(fields []models.FieldSpec, name string) bool {
	for _, field := range fields {
		if field.Name == name {
			return true
		}
	}
	return false
}

func handlerBindMode(ep models.Endpoint) string {
	for _, field := range ep.Request.Fields {
		source := fieldSource(ep, field)
		if source == "" || source == "body" || source == "query" {
			if strings.ToUpper(ep.Method) == "GET" || source == "query" {
				return "query"
			}
			return "json"
		}
	}
	return "none"
}

func endpointNeedsRawBody(ep models.Endpoint) bool {
	for _, field := range ep.Request.Fields {
		if fieldSource(ep, field) == "raw_body" {
			return true
		}
	}
	return false
}

func fieldSource(ep models.Endpoint, field models.FieldSpec) string {
	if field.Source != "" {
		return field.Source
	}

	paramPattern := regexp.MustCompile(`:([A-Za-z0-9_]+)`)
	matches := paramPattern.FindAllStringSubmatch(ep.Path, -1)
	for _, match := range matches {
		if len(match) >= 2 && match[1] == field.Name {
			return "path"
		}
	}

	return ""
}
