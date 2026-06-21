package cliapp

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestBusinessFlowsDoNotOwnPromptInput 固定业务层只能使用 pkg/cli 的 UI 抽象，不恢复自建输入循环。
func TestBusinessFlowsDoNotOwnPromptInput(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(file)
	forbidden := []string{
		"promptReader",
		"newPromptReader",
		"readLine",
		"RunHome",
		"RunStartWizard",
		"RunServiceMenu",
		"PromptConfigPath",
		"PromptAndPersistPrivacy",
		"PromptInitializationInput",
		"ctx.Stdin",
		".Stdin",
		"os.Stdin",
		"bufio.NewReader",
		"fmt.Fscan",
		"Scanln",
	}

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") || isCLICommandAdapterFile(path) {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		content := string(raw)
		for _, token := range forbidden {
			if strings.Contains(content, token) {
				t.Errorf("%s contains forbidden business prompt token %q", filepath.Base(path), token)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk cliapp sources: %v", err)
	}
}

func isCLICommandAdapterFile(path string) bool {
	base := filepath.Base(path)
	if base == "init_handler.go" || base == "iam_bootstrap_handler.go" {
		return true
	}
	return strings.HasSuffix(base, "_command.go") || base == "system_center_commands.go" || base == "app.go"
}
