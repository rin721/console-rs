package main

import (
	"testing"

	httptransport "github.com/rei0721/go-scaffold/internal/transport/http"
	"gopkg.in/yaml.v3"
)

func TestGeneratedOpenAPIYAMLParsesAndIncludesMainRoutes(t *testing.T) {
	raw, err := httptransport.GenerateOpenAPIYAML()
	if err != nil {
		t.Fatalf("generate openapi.yaml: %v", err)
	}
	var spec map[string]any
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("generated openapi.yaml is not valid YAML: %v", err)
	}
	paths, ok := spec["paths"].(map[string]any)
	if !ok {
		t.Fatalf("generated openapi.yaml missing paths object: %#v", spec["paths"])
	}
	for _, path := range []string{
		"/health",
		"/ready",
		httptransport.OpenAPIPath,
		"/api/v1/setup/status",
		"/api/v1/auth/login",
		"/api/v1/plugins",
		"/api/v1/system/apis",
		"/api/v1/system/server-metrics/history",
	} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("generated openapi.yaml missing path %s", path)
		}
	}
}
