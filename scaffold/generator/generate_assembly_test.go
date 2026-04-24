package generator

import (
	"bytes"
	"go/format"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"wechat-clone/scaffold/models"
)

func TestAssemblyBuilderTemplateFormats(t *testing.T) {
	tmpl, err := template.ParseFiles(filepath.Join("..", "template", "assembly_builder.tmpl"))
	if err != nil {
		t.Fatalf("parse template failed: %v", err)
	}

	cases := []struct {
		name     string
		kind     models.AssemblyKind
		function string
		calls    string
		imported string
	}{
		{
			name:     "http",
			kind:     models.AssemblyKindHTTP,
			function: "BuildHTTPServer",
			calls:    "buildHTTPServer(ctx, appContext)",
			imported: `"wechat-clone/core/shared/transport/http"`,
		},
		{
			name:     "messaging",
			kind:     models.AssemblyKindMessaging,
			function: "BuildMessagingRuntime",
			calls:    "buildMessagingRuntime(cfg, appContext)",
			imported: `"wechat-clone/core/shared/runtime"`,
		},
		{
			name:     "projection",
			kind:     models.AssemblyKindProjection,
			function: "BuildProjectionRuntime",
			calls:    "buildProjectionRuntime(cfg, appContext)",
			imported: `"wechat-clone/core/shared/runtime"`,
		},
		{
			name:     "task",
			kind:     models.AssemblyKindTask,
			function: "BuildTaskRuntime",
			calls:    "buildTaskRuntime(cfg, appContext)",
			imported: `"wechat-clone/core/shared/runtime"`,
		},
		{
			name:     "cron",
			kind:     models.AssemblyKindCron,
			function: "BuildCronRuntime",
			calls:    "buildCronRuntime(cfg, appContext)",
			imported: `"wechat-clone/core/shared/runtime"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, buildAssemblyTemplateData(tc.kind)); err != nil {
				t.Fatalf("execute template failed: %v", err)
			}

			formatted, err := format.Source(buf.Bytes())
			if err != nil {
				t.Fatalf("format output failed: %v\n%s", err, buf.String())
			}

			output := string(formatted)
			if !strings.Contains(output, "func "+tc.function+"(") {
				t.Fatalf("expected function %s, got:\n%s", tc.function, output)
			}
			if !strings.Contains(output, tc.calls) {
				t.Fatalf("expected wrapper call %s, got:\n%s", tc.calls, output)
			}
			if !strings.Contains(output, tc.imported) {
				t.Fatalf("expected import %s, got:\n%s", tc.imported, output)
			}
		})
	}
}
