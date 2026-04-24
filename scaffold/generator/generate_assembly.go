package generator

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"text/template"

	"wechat-clone/scaffold/models"
)

type assemblyTemplateData struct {
	FunctionName     string
	ImplFunctionName string
	Signature        string
	Arguments        string
	ReturnType       string
	UsesContext      bool
	UsesConfig       bool
	IsHTTP           bool
}

func GenerateAssembly(spec *models.AssemblySpec) (string, error) {
	if spec == nil {
		return "", errors.New("assembly spec is nil")
	}
	if len(spec.Modules) == 0 {
		return "", errors.New("no assembly modules to generate")
	}

	tmpl, err := template.ParseFiles("scaffold/template/assembly_builder.tmpl")
	if err != nil {
		return "", err
	}

	created := 0
	updated := 0
	skipped := 0

	for _, module := range spec.Modules {
		for _, kind := range module.Kinds {
			dst := assemblyTargetPath(module.Name, kind)
			if fileExists(dst) && !isGeneratedFile(dst, "assembly") {
				skipped++
				continue
			}
			if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
				return "", err
			}

			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, buildAssemblyTemplateData(kind)); err != nil {
				return "", err
			}

			formatted, err := format.Source(buf.Bytes())
			if err != nil {
				return "", fmt.Errorf("format assembly builder failed: %w", err)
			}

			alreadyExists := fileExists(dst)
			if err := os.WriteFile(dst, formatted, 0o644); err != nil {
				return "", err
			}

			if alreadyExists {
				updated++
			} else {
				created++
			}
		}
	}

	return fmt.Sprintf("generated %d assembly builder(s), updated %d existing generated file(s), skipped %d hand-written file(s)", created, updated, skipped), nil
}

func buildAssemblyTemplateData(kind models.AssemblyKind) assemblyTemplateData {
	switch kind {
	case models.AssemblyKindHTTP:
		return assemblyTemplateData{
			FunctionName:     "BuildHTTPServer",
			ImplFunctionName: "buildHTTPServer",
			Signature:        "ctx context.Context, appContext *appCtx.AppContext",
			Arguments:        "ctx, appContext",
			ReturnType:       "infrahttp.HTTPServer",
			UsesContext:      true,
			IsHTTP:           true,
		}
	case models.AssemblyKindMessaging:
		return assemblyTemplateData{
			FunctionName:     "BuildMessagingRuntime",
			ImplFunctionName: "buildMessagingRuntime",
			Signature:        "cfg *config.Config, appContext *appCtx.AppContext",
			Arguments:        "cfg, appContext",
			ReturnType:       "modruntime.Module",
			UsesConfig:       true,
		}
	case models.AssemblyKindProjection:
		return assemblyTemplateData{
			FunctionName:     "BuildProjectionRuntime",
			ImplFunctionName: "buildProjectionRuntime",
			Signature:        "cfg *config.Config, appContext *appCtx.AppContext",
			Arguments:        "cfg, appContext",
			ReturnType:       "modruntime.Module",
			UsesConfig:       true,
		}
	case models.AssemblyKindTask:
		return assemblyTemplateData{
			FunctionName:     "BuildTaskRuntime",
			ImplFunctionName: "buildTaskRuntime",
			Signature:        "cfg *config.Config, appContext *appCtx.AppContext",
			Arguments:        "cfg, appContext",
			ReturnType:       "modruntime.Module",
			UsesConfig:       true,
		}
	case models.AssemblyKindCron:
		return assemblyTemplateData{
			FunctionName:     "BuildCronRuntime",
			ImplFunctionName: "buildCronRuntime",
			Signature:        "cfg *config.Config, appContext *appCtx.AppContext",
			Arguments:        "cfg, appContext",
			ReturnType:       "modruntime.Module",
			UsesConfig:       true,
		}
	default:
		return assemblyTemplateData{}
	}
}

func assemblyTargetPath(moduleName string, kind models.AssemblyKind) string {
	return filepath.Join("core", "modules", moduleName, "assembly", string(kind)+"_builder.go")
}
