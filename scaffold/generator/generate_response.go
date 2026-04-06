package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go-socket/scaffold/models"
	"go-socket/scaffold/utils"
	"go/format"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

func GenerateResponse(endpoints []models.Endpoint) (string, error) {
	tmpl, err := template.ParseFiles("scaffold/template/response.tmpl")
	if err != nil {
		return "", err
	}
	if len(endpoints) == 0 {
		return "", errors.New("no endpoints to generate response")
	}

	seen := make(map[string]bool)
	created := 0
	skipped := 0
	for _, ep := range endpoints {
		if ep.Response.Struct == "" {
			continue
		}

		module, err := moduleForUsecase(ep.Usecase.Name)
		if err != nil {
			return "", err
		}
		key := module.FsRoot + ":out:" + ep.Response.Struct
		if seen[key] {
			continue
		}
		seen[key] = true

		fileName := utils.Snake(ep.Response.Struct) + "_response.go"
		dst := filepath.Join(module.FsRoot, "application/dto/out", fileName)
		if structExistsInDirExcept(filepath.Dir(dst), ep.Response.Struct, dst) {
			if fileExists(dst) && isGeneratedFile(dst, "response") {
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
		if fileExists(dst) && !isGeneratedFile(dst, "response") {
			skipped++
			continue
		}

		data := responseTemplateData{
			PackageName:       "out",
			StructName:        ep.Response.Struct,
			Fields:            mapResponseFields(ep.Response.Fields),
			AdditionalStructs: filterNestedStructs(filepath.Dir(dst), dst, mapNestedStructs(ep.Response.Fields)),
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return "", err
		}
		formatted, err := format.Source(buf.Bytes())
		if err != nil {
			return "", fmt.Errorf("format response DTO failed: %v", err)
		}
		if err := os.WriteFile(dst, formatted, 0o644); err != nil {
			return "", err
		}
		created++
	}

	return fmt.Sprintf("generated %d response DTO(s), skipped %d existing file(s)", created, skipped), nil
}

type responseTemplateData struct {
	PackageName       string
	StructName        string
	Fields            []responseField
	AdditionalStructs []nestedStruct
}

type responseField struct {
	GoName   string
	Type     string
	JSONName string
}

type nestedStruct struct {
	StructName string
	Fields     []responseField
}

func mapResponseFields(fields []models.FieldSpec) []responseField {
	result := make([]responseField, 0, len(fields))
	for _, f := range fields {
		fieldType := responseFieldType(f)
		result = append(result, responseField{
			GoName:   utils.Pascal(f.Name),
			Type:     fieldType,
			JSONName: f.Name,
		})
	}
	return result
}

func mapNestedStructs(fields []models.FieldSpec) []nestedStruct {
	result := make([]nestedStruct, 0)
	seen := make(map[string]bool)
	for _, f := range fields {
		if f.Type != "array" || f.Items == nil || f.Items.Struct == "" {
			continue
		}
		if seen[f.Items.Struct] {
			continue
		}
		seen[f.Items.Struct] = true
		result = append(result, nestedStruct{
			StructName: f.Items.Struct,
			Fields:     mapResponseFields(f.Items.Fields),
		})
	}
	return result
}

func responseFieldType(field models.FieldSpec) string {
	switch strings.ToLower(strings.TrimSpace(field.Type)) {
	case "array":
		if field.Items != nil && field.Items.Struct != "" {
			return "[]" + field.Items.Struct
		}
		return "[]string"
	case "object":
		if strings.TrimSpace(field.Struct) != "" {
			return "*" + field.Struct
		}
		return "map[string]interface{}"
	default:
		goType := utils.GoType(field.Type)
		if field.Pointer {
			goType = "*" + goType
		}
		return goType
	}
}

func filterNestedStructs(dir, dst string, structs []nestedStruct) []nestedStruct {
	filtered := make([]nestedStruct, 0, len(structs))
	for _, nested := range structs {
		if structExistsInDirExcept(dir, nested.StructName, dst) {
			continue
		}
		filtered = append(filtered, nested)
	}
	return filtered
}
