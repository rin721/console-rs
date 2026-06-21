package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rei0721/go-scaffold/pkg/cli"
)

func TestOpenAPICommandWritesStdout(t *testing.T) {
	var stdout bytes.Buffer
	spec := newOpenAPICommand()
	if err := spec.Run(&cli.Context{
		Flags:  map[string]interface{}{"output": ""},
		Stdout: &stdout,
	}); err != nil {
		t.Fatalf("run openapi command: %v", err)
	}
	if !strings.Contains(stdout.String(), `"openapi": "3.0.3"`) {
		t.Fatalf("expected generated openapi on stdout, got %s", stdout.String())
	}
}

func TestOpenAPICommandWritesOutputFile(t *testing.T) {
	output := filepath.Join(t.TempDir(), "docs", "api", "openapi.yaml")
	spec := newOpenAPICommand()
	if err := spec.Run(&cli.Context{
		Flags: map[string]interface{}{"output": output},
	}); err != nil {
		t.Fatalf("run openapi command: %v", err)
	}
	raw, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("read generated output: %v", err)
	}
	if !strings.Contains(string(raw), `"openapi": "3.0.3"`) {
		t.Fatalf("expected generated openapi file, got %s", string(raw))
	}
}
