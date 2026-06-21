package pluginapi_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestPluginAPIDoesNotImportInternalPackages(t *testing.T) {
	files, err := goFilesUnder(".")
	if err != nil {
		t.Fatalf("collect pluginapi files: %v", err)
	}
	for _, file := range files {
		if strings.HasSuffix(filepath.ToSlash(file), "_test.go") {
			continue
		}
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parse %s imports: %v", file, err)
		}
		for _, spec := range parsed.Imports {
			path := strings.Trim(spec.Path.Value, `"`)
			if strings.Contains(path, "/internal/") {
				t.Fatalf("pkg/pluginapi must not import internal package %q from %s", path, file)
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
