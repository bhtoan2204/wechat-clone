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
	"go-socket/scaffold/utils"
)

func GenerateRequest(endpoints []models.Endpoint) (string, error) {
	tmpl, err := template.ParseFiles("scaffold/template/request.tmpl")
	if err != nil {
		return "", err
	}
	if len(endpoints) == 0 {
		return "", errors.New("no endpoints to generate request")
	}

	seen := make(map[string]bool)
	created := 0
	skipped := 0
	for _, ep := range endpoints {
		if ep.Request.Struct == "" {
			continue
		}

		module, err := moduleForUsecase(ep.Usecase.Name)
		if err != nil {
			return "", err
		}
		key := module.FsRoot + ":in:" + ep.Request.Struct
		if seen[key] {
			continue
		}
		seen[key] = true

		fileName := utils.Snake(ep.Request.Struct) + "_request.go"
		dst := filepath.Join(module.FsRoot, "application/dto/in", fileName)
		if structExistsInDirExcept(filepath.Dir(dst), ep.Request.Struct, dst) {
			if fileExists(dst) && isGeneratedFile(dst, "request") {
				if err := os.Remove(dst); err != nil {
					return "", err
				}
			}
			skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}
		if fileExists(dst) && !isGeneratedFile(dst, "request") {
			skipped++
			continue
		}

		fields := mapRequestFields(ep.Request.Fields)
		data := requestTemplateData{
			PackageName:       "in",
			StructName:        ep.Request.Struct,
			Fields:            fields,
			AdditionalStructs: mapRequestNestedStructs(filepath.Dir(dst), dst, ep.Request.Fields),
			NeedsErrors:       requestNeedsErrors(fields),
			NeedsStrings:      requestNeedsStrings(fields),
			HasNormalize:      requestHasNormalize(fields),
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			return "", fmt.Errorf("format request DTO failed: %v", err)
		}
		if err := os.WriteFile(dst, formatted, 0o644); err != nil {
			return "", err
		}
		created++
	}

	return fmt.Sprintf("generated %d request DTO(s), skipped %d existing file(s)", created, skipped), nil
}

type requestTemplateData struct {
	PackageName       string
	StructName        string
	Fields            []requestField
	AdditionalStructs []requestNestedStruct
	NeedsErrors       bool
	NeedsStrings      bool
	HasNormalize      bool
}

type requestField struct {
	GoName         string
	Type           string
	JSONName       string
	BindingTag     string
	Required       bool
	ZeroCheck      string
	NormalizeLines []string
}

type requestNestedStruct struct {
	StructName string
	Fields     []requestField
}

func mapRequestFields(fields []models.FieldSpec) []requestField {
	result := make([]requestField, 0, len(fields))
	for _, f := range fields {
		goName := utils.Pascal(f.Name)
		goType := requestFieldType(f)

		binding := ""
		if f.Required {
			if strings.Contains(strings.ToLower(f.Name), "email") {
				binding = "required,email"
			} else {
				binding = "required"
			}
		}

		result = append(result, requestField{
			GoName:         goName,
			Type:           goType,
			JSONName:       f.Name,
			BindingTag:     binding,
			Required:       f.Required,
			ZeroCheck:      utils.ZeroCheck(goType, goName),
			NormalizeLines: requestNormalizeLines(goType, goName),
		})
	}
	return result
}

func mapRequestNestedStructs(dir, dst string, fields []models.FieldSpec) []requestNestedStruct {
	result := make([]requestNestedStruct, 0)
	seen := make(map[string]bool)
	for _, f := range fields {
		if strings.ToLower(strings.TrimSpace(f.Type)) != "array" || f.Items == nil || f.Items.Struct == "" || len(f.Items.Fields) == 0 {
			continue
		}
		if seen[f.Items.Struct] {
			continue
		}
		if structExistsInDirExcept(dir, f.Items.Struct, dst) {
			continue
		}
		seen[f.Items.Struct] = true
		result = append(result, requestNestedStruct{
			StructName: f.Items.Struct,
			Fields:     mapRequestFields(f.Items.Fields),
		})
	}
	return result
}

func requestNeedsErrors(fields []requestField) bool {
	for _, field := range fields {
		if field.Required && field.ZeroCheck != "" {
			return true
		}
	}
	return false
}

func requestNeedsStrings(fields []requestField) bool {
	for _, field := range fields {
		if len(field.NormalizeLines) > 0 {
			return true
		}
	}
	return false
}

func requestHasNormalize(fields []requestField) bool {
	return requestNeedsStrings(fields)
}

func requestFieldType(field models.FieldSpec) string {
	switch strings.ToLower(strings.TrimSpace(field.Type)) {
	case "array":
		if field.Items != nil && field.Items.Struct != "" {
			return "[]" + field.Items.Struct
		}
		return "[]string"
	case "object":
		if strings.TrimSpace(field.Struct) != "" {
			return field.Struct
		}
		return "map[string]string"
	default:
		goType := utils.GoType(field.Type)
		if field.Pointer {
			goType = "*" + goType
		}
		return goType
	}
}

func requestNormalizeLines(goType, goName string) []string {
	switch goType {
	case "string":
		return []string{fmt.Sprintf("r.%s = strings.TrimSpace(r.%s)", goName, goName)}
	case "*string":
		return []string{
			fmt.Sprintf("if r.%s != nil {", goName),
			fmt.Sprintf("\t*r.%s = strings.TrimSpace(*r.%s)", goName, goName),
			"}",
		}
	case "[]string":
		return []string{
			fmt.Sprintf("for i := range r.%s {", goName),
			fmt.Sprintf("\tr.%s[i] = strings.TrimSpace(r.%s[i])", goName, goName),
			"}",
		}
	case "map[string]string":
		return []string{
			fmt.Sprintf("for key, value := range r.%s {", goName),
			fmt.Sprintf("\tr.%s[key] = strings.TrimSpace(value)", goName),
			"}",
		}
	default:
		return nil
	}
}
