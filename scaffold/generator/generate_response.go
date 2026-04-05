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

		data := responseTemplateData{
			PackageName:       "out",
			StructName:        ep.Response.Struct,
			Fields:            mapResponseFields(ep.Response.Fields),
			AdditionalStructs: mapNestedStructs(ep.Response.Fields),
		}

		fileName := utils.Snake(ep.Response.Struct) + "_response.go"
		dst := filepath.Join(module.FsRoot, "application/dto/out", fileName)
		if structExistsInDir(filepath.Dir(dst), ep.Response.Struct) {
			skipped++
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return "", err
		}
		if fileExists(dst) {
			skipped++
			continue
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
		fieldType := utils.GoType(f.Type)
		if f.Type == "array" && f.Items != nil && f.Items.Struct != "" {
			fieldType = "[]" + f.Items.Struct
		}
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
