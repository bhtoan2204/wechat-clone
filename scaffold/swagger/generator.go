package swagger

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go-socket/scaffold/models"
)

var pathParamPattern = regexp.MustCompile(`:([A-Za-z0-9_]+)`)

func GenerateDefault() (*GeneratedSpec, error) {
	return Generate(DefaultSpecDir, DefaultOutputDir)
}

func Generate(specDir, outputDir string) (*GeneratedSpec, error) {
	spec, err := models.LoadAPISpecDir(specDir)
	if err != nil {
		return nil, err
	}

	document, err := BuildDocument(spec)
	if err != nil {
		return nil, err
	}

	payload, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal openapi json failed: %w", err)
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create swagger output dir failed: %w", err)
	}

	outputPath := filepath.Join(outputDir, DefaultOutputFile)
	if err := os.WriteFile(outputPath, payload, 0o644); err != nil {
		return nil, fmt.Errorf("write swagger json failed: %w", err)
	}

	absOutputPath, err := filepath.Abs(outputPath)
	if err != nil {
		absOutputPath = outputPath
	}

	return &GeneratedSpec{
		Document:   document,
		JSON:       payload,
		OutputPath: absOutputPath,
	}, nil
}

func BuildDocument(spec *models.APISpec) (*Document, error) {
	if spec == nil {
		return nil, fmt.Errorf("api spec is required")
	}

	doc := &Document{
		OpenAPI: defaultOpenAPIVer,
		Info: Info{
			Title:       defaultTitle,
			Version:     fmt.Sprintf("v%d", spec.Version),
			Description: defaultDescription,
		},
		Servers: []Server{{URL: strings.TrimSpace(spec.BasePath)}},
		Paths:   make(Paths),
		Components: Components{
			Schemas: make(map[string]*Schema),
		},
	}

	if doc.Servers[0].URL == "" {
		doc.Servers = nil
	}

	hasAuthEndpoint := false
	for _, ep := range spec.Endpoints {
		moduleTag := endpointTag(ep)
		if moduleTag == "" {
			moduleTag = "default"
		}

		registerPayloadSchemas(doc.Components.Schemas, componentName(moduleTag, ep.Request.Struct), ep.Request)
		registerPayloadSchemas(doc.Components.Schemas, componentName(moduleTag, ep.Response.Struct), ep.Response)

		if ep.Auth {
			hasAuthEndpoint = true
		}

		pathKey := normalizeEndpointPath(ep.Path)
		if _, exists := doc.Paths[pathKey]; !exists {
			doc.Paths[pathKey] = &PathItem{}
		}

		operation := buildOperation(ep, moduleTag)
		switch strings.ToUpper(strings.TrimSpace(ep.Method)) {
		case "GET":
			doc.Paths[pathKey].Get = operation
		case "POST":
			doc.Paths[pathKey].Post = operation
		case "PUT":
			doc.Paths[pathKey].Put = operation
		case "PATCH":
			doc.Paths[pathKey].Patch = operation
		case "DELETE":
			doc.Paths[pathKey].Delete = operation
		default:
			return nil, fmt.Errorf("unsupported http method %q for endpoint %s", ep.Method, ep.Name)
		}
	}

	if hasAuthEndpoint {
		doc.Components.SecuritySchemes = map[string]*SecurityScheme{
			"BearerAuth": {
				Type:         "http",
				Scheme:       "bearer",
				BearerFormat: "JWT",
				Description:  "Bearer access token",
			},
		}
	}

	return doc, nil
}

func buildOperation(ep models.Endpoint, moduleTag string) *Operation {
	operation := &Operation{
		Tags:        []string{moduleTag},
		Summary:     ep.Name,
		OperationID: ep.Name,
		Parameters:  buildParameters(ep),
		Responses: map[string]*Response{
			fmt.Sprintf("%d", successStatus(ep)): {
				Description: fmt.Sprintf("%s response", ep.Name),
				Content: map[string]*MediaType{
					"application/json": {
						Schema: schemaRef(componentName(moduleTag, ep.Response.Struct)),
					},
				},
			},
		},
	}

	if ep.Auth {
		operation.Security = []map[string][]string{
			{"BearerAuth": {}},
		}
	}

	if requestBody := buildRequestBody(ep, moduleTag); requestBody != nil {
		operation.RequestBody = requestBody
	}

	if len(operation.Parameters) == 0 {
		operation.Parameters = nil
	}

	return operation
}

func buildParameters(ep models.Endpoint) []*Parameter {
	params := make([]*Parameter, 0)
	seen := make(map[string]bool)

	for _, field := range ep.Request.Fields {
		source := fieldLocation(ep, field)
		if source != "path" && source != "query" && source != "header" {
			continue
		}

		name := field.Name
		if source == "header" && strings.TrimSpace(field.Header) != "" {
			name = field.Header
		}
		key := source + ":" + name
		if seen[key] {
			continue
		}
		seen[key] = true

		params = append(params, &Parameter{
			Name:        name,
			In:          source,
			Required:    source == "path" || field.Required,
			Description: fmt.Sprintf("%s parameter", field.Name),
			Schema:      fieldSchemaForPrefix(endpointTag(ep), field),
		})
	}

	for _, match := range pathParamPattern.FindAllStringSubmatch(ep.Path, -1) {
		if len(match) < 2 {
			continue
		}
		name := match[1]
		key := "path:" + name
		if seen[key] {
			continue
		}
		seen[key] = true

		params = append(params, &Parameter{
			Name:        name,
			In:          "path",
			Required:    true,
			Description: fmt.Sprintf("%s parameter", name),
			Schema:      &Schema{Type: "string"},
		})
	}

	sort.Slice(params, func(i, j int) bool {
		if params[i].In == params[j].In {
			return params[i].Name < params[j].Name
		}
		return params[i].In < params[j].In
	})

	return params
}

func buildRequestBody(ep models.Endpoint, moduleTag string) *RequestBody {
	bodyFields := make([]models.FieldSpec, 0)
	rawBodyFields := make([]models.FieldSpec, 0)

	for _, field := range ep.Request.Fields {
		switch fieldLocation(ep, field) {
		case "body":
			bodyFields = append(bodyFields, field)
		case "raw_body":
			rawBodyFields = append(rawBodyFields, field)
		}
	}

	switch {
	case len(bodyFields) > 0:
		return &RequestBody{
			Required: hasRequiredField(bodyFields),
			Content: map[string]*MediaType{
				"application/json": {
					Schema: schemaRef(componentName(moduleTag, ep.Request.Struct)),
				},
			},
		}
	case len(rawBodyFields) > 0:
		return &RequestBody{
			Required: hasRequiredField(rawBodyFields),
			Content: map[string]*MediaType{
				"application/octet-stream": {
					Schema: &Schema{
						Type:   "string",
						Format: "binary",
					},
				},
			},
		}
	default:
		return nil
	}
}

func registerPayloadSchemas(dst map[string]*Schema, name string, payload models.Payload) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(payload.Struct) == "" {
		return
	}
	if _, exists := dst[name]; exists {
		return
	}

	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	required := make([]string, 0)
	for _, field := range payload.Fields {
		schema.Properties[field.Name] = fieldSchemaForPrefix(name, field)
		if field.Required {
			required = append(required, field.Name)
		}
		if field.Items != nil && field.Items.Struct != "" {
			registerPayloadSchemas(dst, componentName(name, field.Items.Struct), *field.Items)
		}
	}

	if len(required) > 0 {
		sort.Strings(required)
		schema.Required = required
	}
	if len(schema.Properties) == 0 {
		schema.AdditionalProperties = true
	}

	dst[name] = schema
}

func fieldSchemaForPrefix(prefix string, field models.FieldSpec) *Schema {
	switch strings.ToLower(strings.TrimSpace(field.Type)) {
	case "string":
		return &Schema{Type: "string"}
	case "bool", "boolean":
		return &Schema{Type: "boolean"}
	case "int":
		return &Schema{Type: "integer", Format: "int32"}
	case "int32":
		return &Schema{Type: "integer", Format: "int32"}
	case "int64":
		return &Schema{Type: "integer", Format: "int64"}
	case "float", "float32":
		return &Schema{Type: "number", Format: "float"}
	case "float64", "double":
		return &Schema{Type: "number", Format: "double"}
	case "array":
		items := &Schema{Type: "string"}
		if field.Items != nil {
			if field.Items.Struct != "" {
				items = schemaRef(componentName(prefix, field.Items.Struct))
			} else {
				items = payloadSchema(prefix, *field.Items)
			}
		}
		return &Schema{
			Type:  "array",
			Items: items,
		}
	case "object":
		return &Schema{
			Type:                 "object",
			AdditionalProperties: true,
		}
	default:
		return &Schema{Type: "string"}
	}
}

func payloadSchema(prefix string, payload models.Payload) *Schema {
	schema := &Schema{
		Type:       "object",
		Properties: make(map[string]*Schema),
	}

	required := make([]string, 0)
	for _, field := range payload.Fields {
		schema.Properties[field.Name] = fieldSchemaForPrefix(prefix, field)
		if field.Required {
			required = append(required, field.Name)
		}
	}
	if len(required) > 0 {
		sort.Strings(required)
		schema.Required = required
	}
	if len(schema.Properties) == 0 {
		schema.AdditionalProperties = true
	}
	return schema
}

func successStatus(ep models.Endpoint) int {
	if ep.SuccessStatus > 0 {
		return ep.SuccessStatus
	}
	return 200
}

func hasRequiredField(fields []models.FieldSpec) bool {
	for _, field := range fields {
		if field.Required {
			return true
		}
	}
	return false
}

func normalizeEndpointPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return "/"
	}
	return pathParamPattern.ReplaceAllString(path, `{$1}`)
}

func endpointTag(ep models.Endpoint) string {
	path := strings.TrimPrefix(strings.TrimSpace(ep.Path), "/")
	if path == "" {
		return ""
	}
	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		return ""
	}
	return endpointLikeTag(parts[0])
}

func endpointLikeTag(value string) string {
	value = strings.TrimSpace(strings.ReplaceAll(value, "-", "_"))
	value = strings.Trim(value, "{}:")
	if value == "" {
		return "default"
	}
	return strings.ToLower(value)
}

func componentName(prefix, structName string) string {
	prefix = pascalize(strings.Trim(strings.ReplaceAll(prefix, "-", "_"), "_ "))
	structName = strings.TrimSpace(structName)
	if structName == "" {
		return prefix
	}
	if prefix == "" {
		return structName
	}
	return prefix + "_" + structName
}

func schemaRef(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + name}
}

func fieldLocation(ep models.Endpoint, field models.FieldSpec) string {
	if field.Source != "" {
		return strings.ToLower(strings.TrimSpace(field.Source))
	}

	for _, match := range pathParamPattern.FindAllStringSubmatch(ep.Path, -1) {
		if len(match) >= 2 && match[1] == field.Name {
			return "path"
		}
	}

	switch strings.ToUpper(strings.TrimSpace(ep.Method)) {
	case "GET":
		return "query"
	default:
		return "body"
	}
}

func pascalize(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}

	parts := strings.FieldsFunc(value, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for idx, part := range parts {
		if part == "" {
			continue
		}
		parts[idx] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "_")
}
