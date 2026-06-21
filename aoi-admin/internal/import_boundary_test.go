package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const modulePath = "github.com/rei0721/go-scaffold"

func TestInternalPackagesDoNotImportThirdPartyInfrastructure(t *testing.T) {
	files, err := goFilesUnder(".")
	if err != nil {
		t.Fatalf("collect internal go files: %v", err)
	}

	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s imports: %v", file, err)
		}

		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if isThirdPartyImport(path) {
				t.Fatalf("internal package must use pkg anti-corruption wrappers instead of third-party import %q from %s", path, file)
			}
		}
	}
}

func TestInternalProductionCodeDoesNotImportPkgOutsideAppAndConfig(t *testing.T) {
	files, err := goFilesUnder(".")
	if err != nil {
		t.Fatalf("collect internal go files: %v", err)
	}

	for _, file := range files {
		normalized := filepath.ToSlash(file)
		if strings.HasSuffix(normalized, "_test.go") || internalPkgImportAllowed(normalized) {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s imports: %v", file, err)
		}

		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if strings.HasPrefix(path, modulePath+"/pkg/") {
				t.Fatalf("internal production code outside app/config must depend on internal ports instead of pkg import %q from %s", path, file)
			}
			if forbiddenPluginExampleImport(path) {
				t.Fatalf("production code must not import plugin examples %q from %s", path, file)
			}
		}
	}
}

func TestProductionCodeDoesNotImportPluginExamples(t *testing.T) {
	for _, root := range []string{"../cmd", ".", "../pkg", "../types"} {
		files, err := goFilesUnder(root)
		if err != nil {
			t.Fatalf("collect go files under %s: %v", root, err)
		}
		for _, file := range files {
			normalized := filepath.ToSlash(file)
			if strings.HasSuffix(normalized, "_test.go") {
				continue
			}
			parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse %s imports: %v", file, err)
			}
			for _, spec := range parsed.Imports {
				path := strings.Trim(spec.Path.Value, `"`)
				if forbiddenPluginExampleImport(path) {
					t.Fatalf("production code must not import plugin examples %q from %s", path, file)
				}
			}
		}
	}
}

func TestPluginCoreDoesNotDependOnInternalOrRPC(t *testing.T) {
	files, err := goFilesUnder("../pkg/plugin")
	if err != nil {
		t.Fatalf("collect pkg/plugin go files: %v", err)
	}
	for _, file := range files {
		normalized := filepath.ToSlash(file)
		if strings.HasSuffix(normalized, "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s imports: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if strings.HasPrefix(path, modulePath+"/internal/") {
				t.Fatalf("pkg/plugin must not import internal package %q from %s", path, file)
			}
			if path == modulePath+"/pkg/rpcserver" {
				t.Fatalf("pkg/plugin must not depend on RPC implementation %q from %s", path, file)
			}
		}
	}
}

func TestLegacyPluginPackagesAreNotPresent(t *testing.T) {
	for _, path := range []string{"pluginhost", "modules/plugins"} {
		if _, err := os.Stat(path); err == nil {
			t.Fatalf("legacy plugin package %s must not be present", path)
		} else if !os.IsNotExist(err) {
			t.Fatalf("stat legacy plugin package %s: %v", path, err)
		}
	}
}

func TestModuleServiceLayerStaysInfrastructureFree(t *testing.T) {
	files, err := goFilesUnder("modules")
	if err != nil {
		t.Fatalf("collect module go files: %v", err)
	}

	for _, file := range files {
		normalized := filepath.ToSlash(file)
		if strings.HasSuffix(normalized, "_test.go") || !isModuleServiceFile(normalized) {
			continue
		}
		module, ok := moduleNameFromServiceFile(normalized)
		if !ok {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s imports: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			switch path {
			case modulePath + "/internal/modules/" + module + "/repository":
				t.Fatalf("service layer must depend on its own repository interface, not repository implementation import %q from %s", path, file)
			case modulePath + "/internal/ports":
				t.Fatalf("service layer must define minimal local interfaces instead of importing shared infrastructure ports from %s", file)
			}
		}

		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		text := string(content)
		for _, pattern := range []string{"http.Client", "smtp.", "os.Getenv", "database.New", "WithExecutor"} {
			if strings.Contains(text, pattern) {
				t.Fatalf("service layer must not contain infrastructure pattern %q in %s", pattern, file)
			}
		}
	}
}

func goFilesUnder(root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func isModuleServiceFile(path string) bool {
	parts := strings.Split(strings.TrimPrefix(path, "./"), "/")
	return len(parts) >= 4 && parts[0] == "modules" && parts[2] == "service"
}

func moduleNameFromServiceFile(path string) (string, bool) {
	parts := strings.Split(strings.TrimPrefix(path, "./"), "/")
	if len(parts) < 4 || parts[0] != "modules" || parts[2] != "service" {
		return "", false
	}
	return parts[1], true
}

func internalPkgImportAllowed(path string) bool {
	path = strings.TrimPrefix(path, "./")
	return strings.HasPrefix(path, "app/") ||
		strings.HasPrefix(path, "config/") ||
		strings.HasPrefix(path, "plugin/")
}

func isThirdPartyImport(path string) bool {
	if strings.HasPrefix(path, modulePath) {
		return false
	}
	first := path
	if idx := strings.Index(first, "/"); idx >= 0 {
		first = first[:idx]
	}
	return strings.Contains(first, ".")
}

func forbiddenPluginExampleImport(path string) bool {
	return strings.HasPrefix(path, modulePath+"/plugins/") || strings.HasPrefix(path, modulePath+"/_examples/")
}
