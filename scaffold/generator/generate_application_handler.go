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

func GenerateApplicationHandler(endpoints []models.Endpoint) (string, error) {
	tmpl, err := template.ParseFiles("scaffold/template/application_handler.tmpl")
	if err != nil {
		return "", err
	}
	if len(endpoints) == 0 {
		return "", errors.New("no endpoints to generate application handler")
	}

	seen := make(map[string]bool)
	created := 0
	updated := 0
	skipped := 0
	for _, ep := range endpoints {
		if !shouldGenerateApplicationHandler(ep) {
			continue
		}

		module, err := moduleForUsecase(ep.Usecase.Name)
		if err != nil {
			return "", err
		}

		kind := applicationHandlerKind(ep)
		key := module.FsRoot + ":" + kind + ":" + ep.Usecase.Method
		if seen[key] {
			continue
		}
		seen[key] = true

		writeResult, err := writeApplicationHandlerFile(tmpl, module, ep)
		if err != nil {
			return "", err
		}
		switch writeResult {
		case applicationHandlerWriteCreated:
			created++
		case applicationHandlerWriteUpdated:
			updated++
		default:
			skipped++
		}
	}

	return fmt.Sprintf("generated %d application handler(s), updated %d existing generated file(s), skipped %d hand-written file(s)", created, updated, skipped), nil
}

func shouldGenerateApplicationHandler(ep models.Endpoint) bool {
	if ep.Usecase.Name == "" || ep.Usecase.Method == "" {
		return false
	}
	if ep.Request.Struct == "" || ep.Response.Struct == "" {
		return false
	}
	return true
}

type applicationHandlerWriteResult string

const (
	applicationHandlerWriteCreated applicationHandlerWriteResult = "created"
	applicationHandlerWriteUpdated applicationHandlerWriteResult = "updated"
	applicationHandlerWriteSkipped applicationHandlerWriteResult = "skipped"
)

func writeApplicationHandlerFile(tmpl *template.Template, module modulePaths, ep models.Endpoint) (applicationHandlerWriteResult, error) {
	kind := applicationHandlerKind(ep)
	handlerName := ep.Usecase.Method
	structName := lowerFirst(handlerName) + "Handler"

	config, err := applicationHandlerConfigForEndpoint(module, ep)
	if err != nil {
		return applicationHandlerWriteSkipped, err
	}

	fileName := utils.Snake(handlerName) + "_handler.go"
	dst := filepath.Join(module.FsRoot, "application", kind, fileName)
	if structExistsInDirExcept(filepath.Dir(dst), structName, dst) {
		return applicationHandlerWriteSkipped, nil
	}

	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return applicationHandlerWriteSkipped, err
	}
	if fileExists(dst) && !isGeneratedFile(dst, "application-handler") {
		return applicationHandlerWriteSkipped, nil
	}
	if fileExists(dst) && !isScaffoldStubFile(dst) {
		return applicationHandlerWriteSkipped, nil
	}
	data := applicationHandlerTemplateData{
		PackageName:       kind,
		HandlerName:       handlerName,
		StructName:        structName,
		RequestStruct:     ep.Request.Struct,
		ResponseType:      responseType(ep.Response),
		RequestDtoImport:  module.ImportRoot + "/application/dto/in",
		ResponseDtoImport: module.ImportRoot + "/application/dto/out",
		CQRSImport:        "go-socket/core/shared/pkg/cqrs",
		Imports:           config.Imports,
		Params:            config.Params,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return applicationHandlerWriteSkipped, err
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return applicationHandlerWriteSkipped, fmt.Errorf("format application handler failed: %v", err)
	}
	alreadyExists := fileExists(dst)
	if err := os.WriteFile(dst, formatted, 0o644); err != nil {
		return applicationHandlerWriteSkipped, err
	}

	if alreadyExists {
		return applicationHandlerWriteUpdated, nil
	}
	return applicationHandlerWriteCreated, nil
}

type applicationHandlerTemplateData struct {
	PackageName       string
	HandlerName       string
	StructName        string
	RequestStruct     string
	ResponseType      string
	RequestDtoImport  string
	ResponseDtoImport string
	CQRSImport        string
	Imports           []applicationHandlerImport
	Params            []applicationHandlerParam
}

type applicationHandlerImport struct {
	Alias string
	Path  string
}

type applicationHandlerParam struct {
	Name string
	Type string
}

type applicationHandlerConfig struct {
	Imports []applicationHandlerImport
	Params  []applicationHandlerParam
}

func applicationHandlerKind(ep models.Endpoint) string {
	if strings.EqualFold(ep.Method, "GET") {
		return "query"
	}
	return "command"
}

func applicationHandlerConfigForEndpoint(module modulePaths, ep models.Endpoint) (applicationHandlerConfig, error) {
	switch module.ImportRoot {
	case "go-socket/core/modules/account":
		return applicationHandlerConfig{
			Imports: append(commonApplicationHandlerImports(module),
				applicationHandlerImport{Path: module.ImportRoot + "/application/service"},
			),
			Params: append(commonApplicationHandlerParams(),
				applicationHandlerParam{Name: "services", Type: "service.Services"},
			),
		}, nil
	case "go-socket/core/modules/notification":
		return applicationHandlerConfig{
			Imports: commonApplicationHandlerImports(module),
			Params:  commonApplicationHandlerParams(),
		}, nil
	case "go-socket/core/modules/ledger":
		return applicationHandlerConfig{
			Imports: append(commonApplicationHandlerImports(module),
				applicationHandlerImport{Path: module.ImportRoot + "/application/service"},
			),
			Params: append(commonApplicationHandlerParams(),
				applicationHandlerParam{Name: "service", Type: "*service.LedgerService"},
			),
		}, nil
	case "go-socket/core/modules/payment":
		return paymentApplicationHandlerConfig(module, ep), nil
	case "go-socket/core/modules/room":
		return roomApplicationHandlerConfig(module, ep), nil
	default:
		return applicationHandlerConfig{}, fmt.Errorf("unsupported module import root: %s", module.ImportRoot)
	}
}

func paymentApplicationHandlerConfig(module modulePaths, ep models.Endpoint) applicationHandlerConfig {
	if usesPaymentService(ep.Usecase.Method) {
		return applicationHandlerConfig{
			Imports: append(commonApplicationHandlerImports(module),
				applicationHandlerImport{Path: module.ImportRoot + "/application/service"},
			),
			Params: append(commonApplicationHandlerParams(),
				applicationHandlerParam{Name: "service", Type: "*service.PaymentService"},
			),
		}
	}

	return applicationHandlerConfig{
		Imports: commonApplicationHandlerImports(module),
		Params:  commonApplicationHandlerParams(),
	}
}

func roomApplicationHandlerConfig(module modulePaths, ep models.Endpoint) applicationHandlerConfig {
	serviceType := "RoomCommandService"
	paramName := "roomService"
	params := commonApplicationHandlerParams()

	if applicationHandlerKind(ep) == "query" {
		serviceType = "RoomQueryService"
		paramName = "roomQueryService"
		if usesRoomChatQueryService(ep.Usecase.Method) {
			serviceType = "ChatQueryService"
			paramName = "chatService"
		}
		params = []applicationHandlerParam{
			{Name: "appCtx", Type: "*appCtx.AppContext"},
			{Name: "readRepo", Type: "repos.QueryRepos"},
		}
	} else if usesRoomMessageCommandService(ep.Usecase.Method) {
		serviceType = "MessageCommandService"
		paramName = "messageService"
	}

	return applicationHandlerConfig{
		Imports: append(commonApplicationHandlerImports(module),
			applicationHandlerImport{Path: module.ImportRoot + "/application/service"},
		),
		Params: append(params,
			applicationHandlerParam{Name: paramName, Type: "*service." + serviceType},
		),
	}
}

func commonApplicationHandlerImports(module modulePaths) []applicationHandlerImport {
	return []applicationHandlerImport{
		{Alias: "appCtx", Path: "go-socket/core/context"},
		{Alias: "repos", Path: module.ImportRoot + "/domain/repos"},
	}
}

func commonApplicationHandlerParams() []applicationHandlerParam {
	return []applicationHandlerParam{
		{Name: "appCtx", Type: "*appCtx.AppContext"},
		{Name: "baseRepo", Type: "repos.Repos"},
	}
}

func usesPaymentService(method string) bool {
	switch method {
	case "CreatePayment", "ProcessWebhook":
		return true
	default:
		return false
	}
}

func usesRoomChatQueryService(method string) bool {
	switch method {
	case "GetRoom", "ListRooms":
		return false
	default:
		return true
	}
}

func usesRoomMessageCommandService(method string) bool {
	switch method {
	case "SendChatMessage", "EditChatMessage", "DeleteChatMessage", "ForwardChatMessage", "MarkChatMessageStatus", "CreateMessage":
		return true
	default:
		return false
	}
}
