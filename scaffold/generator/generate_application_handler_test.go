package generator

import (
	"bytes"
	"go/format"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
)

func TestApplicationHandlerTemplateFormats(t *testing.T) {
	tmpl, err := template.ParseFiles(filepath.Join("..", "template", "application_handler.tmpl"))
	if err != nil {
		t.Fatalf("parse template failed: %v", err)
	}

	cases := []applicationHandlerTemplateData{
		{
			PackageName:       "command",
			HandlerName:       "CreatePayment",
			StructName:        "createPaymentHandler",
			RequestStruct:     "CreatePaymentRequest",
			ResponseType:      "*out.CreatePaymentResponse",
			RequestDtoImport:  "go-socket/core/modules/payment/application/dto/in",
			ResponseDtoImport: "go-socket/core/modules/payment/application/dto/out",
			CQRSImport:        "go-socket/core/shared/pkg/cqrs",
			Imports: []applicationHandlerImport{
				{Alias: "appCtx", Path: "go-socket/core/context"},
				{Alias: "repos", Path: "go-socket/core/modules/payment/domain/repos"},
				{Path: "go-socket/core/modules/payment/application/service"},
			},
			Params: []applicationHandlerParam{
				{Name: "appCtx", Type: "*appCtx.AppContext"},
				{Name: "baseRepo", Type: "repos.Repos"},
				{Name: "service", Type: "*service.PaymentService"},
			},
		},
		{
			PackageName:       "query",
			HandlerName:       "GetProfile",
			StructName:        "getProfileHandler",
			RequestStruct:     "GetProfileRequest",
			ResponseType:      "*out.GetProfileResponse",
			RequestDtoImport:  "go-socket/core/modules/account/application/dto/in",
			ResponseDtoImport: "go-socket/core/modules/account/application/dto/out",
			CQRSImport:        "go-socket/core/shared/pkg/cqrs",
			Imports: []applicationHandlerImport{
				{Alias: "appCtx", Path: "go-socket/core/context"},
				{Path: "go-socket/core/modules/account/application/service"},
				{Alias: "repos", Path: "go-socket/core/modules/account/domain/repos"},
			},
			Params: []applicationHandlerParam{
				{Name: "appCtx", Type: "*appCtx.AppContext"},
				{Name: "baseRepo", Type: "repos.Repos"},
				{Name: "services", Type: "service.Services"},
			},
		},
		{
			PackageName:       "query",
			HandlerName:       "ListRooms",
			StructName:        "listRoomsHandler",
			RequestStruct:     "ListRoomsRequest",
			ResponseType:      "*out.ListRoomsResponse",
			RequestDtoImport:  "go-socket/core/modules/room/application/dto/in",
			ResponseDtoImport: "go-socket/core/modules/room/application/dto/out",
			CQRSImport:        "go-socket/core/shared/pkg/cqrs",
			Imports: []applicationHandlerImport{
				{Alias: "appCtx", Path: "go-socket/core/context"},
				{Alias: "repos", Path: "go-socket/core/modules/room/domain/repos"},
				{Path: "go-socket/core/modules/room/application/service"},
			},
			Params: []applicationHandlerParam{
				{Name: "appCtx", Type: "*appCtx.AppContext"},
				{Name: "readRepo", Type: "repos.QueryRepos"},
				{Name: "roomQueryService", Type: "*service.RoomQueryService"},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.HandlerName, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, tc); err != nil {
				t.Fatalf("execute template failed: %v", err)
			}

			formatted, err := format.Source(buf.Bytes())
			if err != nil {
				t.Fatalf("format output failed: %v\n%s", err, buf.String())
			}

			output := string(formatted)
			if !strings.Contains(output, `return nil, fmt.Errorf("not implemented yet")`) {
				t.Fatalf("expected scaffold body, got:\n%s", output)
			}
			if !strings.Contains(output, "func New"+tc.HandlerName+"(") {
				t.Fatalf("expected constructor for %s, got:\n%s", tc.HandlerName, output)
			}
			if tc.HandlerName == "CreatePayment" {
				if !strings.Contains(output, "appCtx *appCtx.AppContext") {
					t.Fatalf("expected appCtx param, got:\n%s", output)
				}
				if !strings.Contains(output, "baseRepo repos.Repos") {
					t.Fatalf("expected baseRepo param, got:\n%s", output)
				}
			}
			if tc.HandlerName == "ListRooms" && !strings.Contains(output, "readRepo repos.QueryRepos") {
				t.Fatalf("expected readRepo param, got:\n%s", output)
			}
		})
	}
}
