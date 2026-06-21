package httptransport

import (
	"bytes"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"

	systemmodel "github.com/rei0721/go-scaffold/internal/modules/system/model"
)

func GenerateOpenAPIYAML() ([]byte, error) {
	document := openAPIDocument(MainHTTPContracts())
	var out bytes.Buffer
	writeYAML(&out, document, 0)
	return out.Bytes(), nil
}

func openAPIDocument(contracts []RouteContract) map[string]any {
	paths := make(map[string]any)
	tagSeen := make(map[string]struct{})
	tags := make([]any, 0)
	for _, contract := range contracts {
		if !contract.IncludeOpenAPI {
			continue
		}
		if _, ok := tagSeen[contract.Tag]; !ok {
			tagSeen[contract.Tag] = struct{}{}
			tags = append(tags, map[string]any{"name": contract.Tag})
		}
		path := openAPIPath(contract.Path)
		methods, _ := paths[path].(map[string]any)
		if methods == nil {
			methods = make(map[string]any)
			paths[path] = methods
		}
		methods[strings.ToLower(contract.Method)] = openAPIOperation(contract)
	}
	return map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":       "go-scaffold API",
			"version":     "1.0.0",
			"description": "当前 go-scaffold HTTP 路由自动生成的 API 契约。本文件由 route contract registry 生成，不应手写维护。",
		},
		"servers": []any{
			map[string]any{"url": "http://127.0.0.1:9999", "description": "默认本地服务"},
		},
		"tags":  tags,
		"paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{
				"bearerAuth": map[string]any{"type": "http", "scheme": "bearer", "bearerFormat": "JWT or API token"},
			},
			"schemas": map[string]any{
				"ErrorResult": errorResultSchema(),
			},
		},
	}
}

func openAPIOperation(contract RouteContract) map[string]any {
	operation := map[string]any{
		"tags":        []any{contract.Tag},
		"summary":     contract.Summary,
		"operationId": contract.ID,
		"responses":   openAPIResponses(contract),
	}
	if contract.Description != "" {
		operation["description"] = contract.Description
	}
	if contract.Permission != "" {
		operation["x-permission"] = contract.Permission
	}
	if contract.Scope != "" {
		operation["x-scope"] = contract.Scope
	}
	if contract.Access == systemmodel.APIAccessPublic {
		operation["security"] = []any{}
	}
	params := openAPIParameters(contract)
	if len(params) > 0 {
		operation["parameters"] = params
	}
	if contract.RequestType != nil || contract.RequestContent == ContentMultipart {
		operation["requestBody"] = openAPIRequestBody(contract)
	}
	return operation
}

func openAPIRequestBody(contract RouteContract) map[string]any {
	contentType := contract.RequestContent
	if contentType == "" {
		contentType = ContentJSON
	}
	schema := map[string]any{"type": "object"}
	if contentType == ContentMultipart {
		properties := make(map[string]any)
		required := make([]any, 0)
		for _, param := range contract.Params {
			if param.In != ParamInMultipart {
				continue
			}
			field := primitiveSchema(param.Type, param.Format)
			if param.Name == "file" {
				field["format"] = "binary"
			}
			properties[param.Name] = field
			if param.Required {
				required = append(required, param.Name)
			}
		}
		schema["properties"] = properties
		if len(required) > 0 {
			schema["required"] = required
		}
	} else if contract.RequestType != nil {
		schema = schemaForType(contract.RequestType, map[reflect.Type]bool{})
	}
	return map[string]any{
		"required": true,
		"content": map[string]any{
			contentType: map[string]any{"schema": schema},
		},
	}
}

func openAPIResponses(contract RouteContract) map[string]any {
	status := contract.Status
	if status == 0 {
		status = 200
	}
	contentType := contract.ResponseContent
	if contentType == "" {
		contentType = ContentJSON
	}
	schema := map[string]any{"type": "object", "additionalProperties": true}
	if contract.RawResponse {
		switch contentType {
		case ContentOctet:
			schema = map[string]any{"type": "string", "format": "binary"}
		case ContentYAML:
			schema = map[string]any{"type": "string"}
		}
	} else {
		dataSchema := map[string]any{"type": "object", "additionalProperties": true}
		if contract.ResponseType != nil {
			dataSchema = schemaForType(contract.ResponseType, map[reflect.Type]bool{})
		}
		schema = resultSchema(dataSchema)
	}
	return map[string]any{
		strconv.Itoa(status): map[string]any{
			"description": "OK",
			"content": map[string]any{
				contentType: map[string]any{"schema": schema},
			},
		},
		"400": errorResponse("Bad request"),
		"401": errorResponse("Unauthorized"),
		"403": errorResponse("Forbidden"),
		"404": errorResponse("Not found"),
		"500": errorResponse("Internal server error"),
	}
}

func openAPIParameters(contract RouteContract) []any {
	params := make([]RouteParam, 0, len(contract.Params))
	params = append(params, contract.Params...)
	for _, name := range pathParamNames(contract.Path) {
		if hasParam(params, ParamInPath, name) {
			continue
		}
		params = append(params, pathString(name))
	}
	out := make([]any, 0, len(params))
	for _, param := range params {
		if param.In == ParamInMultipart {
			continue
		}
		required := param.Required || param.In == ParamInPath
		item := map[string]any{
			"name":     param.Name,
			"in":       param.In,
			"required": required,
			"schema":   primitiveSchema(param.Type, param.Format),
		}
		if param.Description != "" {
			item["description"] = param.Description
		}
		out = append(out, item)
	}
	return out
}

func hasParam(params []RouteParam, in string, name string) bool {
	for _, param := range params {
		if param.In == in && param.Name == name {
			return true
		}
	}
	return false
}

func pathParamNames(path string) []string {
	parts := strings.Split(path, "/")
	names := make([]string, 0)
	for _, part := range parts {
		if strings.HasPrefix(part, ":") && len(part) > 1 {
			names = append(names, strings.TrimPrefix(part, ":"))
		}
	}
	return names
}

func openAPIPath(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if strings.HasPrefix(part, ":") && len(part) > 1 {
			parts[i] = "{" + strings.TrimPrefix(part, ":") + "}"
		}
	}
	return strings.Join(parts, "/")
}

func primitiveSchema(typ string, format string) map[string]any {
	if typ == "" {
		typ = "string"
	}
	schema := map[string]any{"type": typ}
	if format != "" {
		schema["format"] = format
	}
	return schema
}

func resultSchema(data map[string]any) map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"code":        map[string]any{"type": "integer", "format": "int32"},
			"messageKey":  map[string]any{"type": "string"},
			"message":     map[string]any{"type": "string"},
			"messageArgs": map[string]any{"type": "object", "additionalProperties": true},
			"data":        data,
			"traceId":     map[string]any{"type": "string"},
			"serverTime":  map[string]any{"type": "integer", "format": "int64"},
		},
	}
}

func errorResultSchema() map[string]any {
	return resultSchema(map[string]any{"type": "object", "additionalProperties": true})
}

func errorResponse(description string) map[string]any {
	return map[string]any{
		"description": description,
		"content": map[string]any{
			ContentJSON: map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/ErrorResult"}},
		},
	}
}

func schemaForType(t reflect.Type, seen map[reflect.Type]bool) map[string]any {
	if t == nil {
		return map[string]any{"type": "object", "additionalProperties": true}
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if t == reflect.TypeOf(time.Time{}) {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	switch t.Kind() {
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32:
		return map[string]any{"type": "integer", "format": "int32"}
	case reflect.Int64:
		return map[string]any{"type": "integer", "format": "int64"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32:
		return map[string]any{"type": "integer", "format": "int32", "minimum": 0}
	case reflect.Uint64:
		return map[string]any{"type": "integer", "format": "int64", "minimum": 0}
	case reflect.Float32, reflect.Float64:
		return map[string]any{"type": "number", "format": "double"}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Slice, reflect.Array:
		itemType := t.Elem()
		if itemType.Kind() == reflect.Uint8 {
			return map[string]any{"type": "string", "format": "binary"}
		}
		return map[string]any{"type": "array", "items": schemaForType(itemType, seen)}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": schemaForType(t.Elem(), seen)}
	case reflect.Interface:
		return map[string]any{"type": "object", "additionalProperties": true}
	case reflect.Struct:
		if seen[t] {
			return map[string]any{"type": "object", "additionalProperties": true}
		}
		seen[t] = true
		defer delete(seen, t)
		properties := make(map[string]any)
		required := make([]any, 0)
		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			if field.PkgPath != "" && !field.Anonymous {
				continue
			}
			name, omitempty, ok := jsonFieldName(field)
			if !ok {
				continue
			}
			properties[name] = schemaForType(field.Type, seen)
			if bindingRequired(field) && !omitempty {
				required = append(required, name)
			}
		}
		schema := map[string]any{"type": "object", "properties": properties}
		if len(required) > 0 {
			schema["required"] = required
		}
		return schema
	default:
		return map[string]any{"type": "object", "additionalProperties": true}
	}
}

func jsonFieldName(field reflect.StructField) (string, bool, bool) {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return "", false, false
	}
	name := field.Name
	omitempty := false
	if tag != "" {
		parts := strings.Split(tag, ",")
		if parts[0] != "" {
			name = parts[0]
		}
		for _, option := range parts[1:] {
			if option == "omitempty" {
				omitempty = true
				break
			}
		}
	}
	return name, omitempty, true
}

func bindingRequired(field reflect.StructField) bool {
	for _, tagName := range []string{"binding", "validate"} {
		for _, item := range strings.Split(field.Tag.Get(tagName), ",") {
			if strings.TrimSpace(item) == "required" {
				return true
			}
		}
	}
	return false
}

func writeYAML(out *bytes.Buffer, value any, indent int) {
	switch v := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(v))
		for key := range v {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			writeIndent(out, indent)
			out.WriteString(yamlScalar(key))
			out.WriteString(":")
			if yamlInline(v[key]) {
				out.WriteByte(' ')
				writeYAML(out, v[key], 0)
			} else {
				out.WriteByte('\n')
				writeYAML(out, v[key], indent+2)
			}
		}
	case []any:
		if len(v) == 0 {
			out.WriteString("[]\n")
			return
		}
		for _, item := range v {
			writeIndent(out, indent)
			out.WriteString("-")
			if yamlInline(item) {
				out.WriteByte(' ')
				writeYAML(out, item, 0)
			} else {
				out.WriteByte('\n')
				writeYAML(out, item, indent+2)
			}
		}
	case string:
		out.WriteString(yamlScalar(v))
		out.WriteByte('\n')
	case bool:
		if v {
			out.WriteString("true\n")
		} else {
			out.WriteString("false\n")
		}
	case int:
		out.WriteString(strconv.Itoa(v))
		out.WriteByte('\n')
	case int64:
		out.WriteString(strconv.FormatInt(v, 10))
		out.WriteByte('\n')
	case float64:
		out.WriteString(strconv.FormatFloat(v, 'f', -1, 64))
		out.WriteByte('\n')
	case nil:
		out.WriteString("null\n")
	default:
		out.WriteString(yamlScalar(fmt.Sprint(v)))
		out.WriteByte('\n')
	}
}

func yamlInline(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		return false
	case []any:
		return len(v) == 0
	default:
		return true
	}
}

func writeIndent(out *bytes.Buffer, indent int) {
	for i := 0; i < indent; i++ {
		out.WriteByte(' ')
	}
}

func yamlScalar(value string) string {
	return strconv.Quote(value)
}
