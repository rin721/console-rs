package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestNamespaceLocalizeResolveAndMissingKeys(t *testing.T) {
	root := t.TempDir()
	writeLocale(t, root, NamespaceUI, "zh-CN", "welcome: 欢迎 {{.Name}}\n")
	writeLocale(t, root, NamespaceUI, "en-US", "welcome: Welcome {{.Name}}\n")
	writeLocale(t, root, NamespaceAPI, "zh-CN", "common.success: 操作成功\n")
	writeLocale(t, root, NamespaceAPI, "en-US", "common.success: Success\n")
	writeLocale(t, root, NamespaceValidation, "zh-CN", "required: 必填\n")
	writeLocale(t, root, NamespaceValidation, "en-US", "required: Required\n")
	writeLocale(t, root, NamespaceSystem, "zh-CN", "brand.productName: 示例\n")
	writeLocale(t, root, NamespaceSystem, "en-US", "brand.productName: Example\n")

	seenMissing := []MissingKey{}
	manager, err := New(&Config{
		DefaultLocale:  "zh-CN",
		FallbackLocale: "zh-CN",
		Supported:      []string{"zh-CN", "en-US"},
		Resources: map[string]string{
			NamespaceUI:         filepath.Join(root, NamespaceUI),
			NamespaceAPI:        filepath.Join(root, NamespaceAPI),
			NamespaceValidation: filepath.Join(root, NamespaceValidation),
			NamespaceSystem:     filepath.Join(root, NamespaceSystem),
		},
		MissingLogger: func(item MissingKey) {
			seenMissing = append(seenMissing, item)
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := manager.ValidateResources(); err != nil {
		t.Fatalf("ValidateResources() error = %v", err)
	}

	if got := manager.ResolveLocale("fr-FR, en-US;q=0.9"); got != "en-US" {
		t.Fatalf("ResolveLocale() = %q, want en-US", got)
	}
	if got := manager.ResolveLocale("en"); got != "en-US" {
		t.Fatalf("ResolveLocale(short) = %q, want en-US", got)
	}
	if got := manager.ResolveLocale("fr-FR"); got != "zh-CN" {
		t.Fatalf("ResolveLocale(fallback) = %q, want zh-CN", got)
	}

	msg := manager.Localize("en-US", NamespaceUI, "welcome", map[string]any{"Name": "Codex"})
	if msg != "Welcome Codex" {
		t.Fatalf("Localize() = %q, want Welcome Codex", msg)
	}

	missing := manager.Localize("en-US", NamespaceUI, "missing.key", nil)
	if missing != "missing.key" {
		t.Fatalf("missing Localize() = %q, want key", missing)
	}
	if len(manager.MissingKeys()) != 1 || len(seenMissing) != 1 {
		t.Fatalf("missing keys not recorded: manager=%v callback=%v", manager.MissingKeys(), seenMissing)
	}
}

func TestValidateResourcesRequiresDefaultNamespaces(t *testing.T) {
	root := t.TempDir()
	writeLocale(t, root, NamespaceUI, "zh-CN", "welcome: 欢迎\n")

	manager, err := New(&Config{
		DefaultLocale:  "zh-CN",
		FallbackLocale: "zh-CN",
		Supported:      []string{"zh-CN"},
		Resources: map[string]string{
			NamespaceUI: filepath.Join(root, NamespaceUI),
		},
	})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if err := manager.ValidateResources(); err == nil {
		t.Fatal("ValidateResources() error = nil, want missing namespace error")
	}
}

func TestRepositoryLocaleResourcesAreCompleteAndReadable(t *testing.T) {
	root := findRepoRoot(t)
	resources := map[string]string{
		NamespaceUI:         filepath.Join(root, "configs", "locales", NamespaceUI),
		NamespaceAPI:        filepath.Join(root, "configs", "locales", NamespaceAPI),
		NamespaceValidation: filepath.Join(root, "configs", "locales", NamespaceValidation),
		NamespaceSystem:     filepath.Join(root, "configs", "locales", NamespaceSystem),
	}
	manager, err := New(&Config{
		DefaultLocale:  DefaultLanguage,
		FallbackLocale: DefaultLanguage,
		Supported:      SupportedLanguagesStringSlice,
		Resources:      resources,
	})
	if err != nil {
		t.Fatalf("load repository locales: %v", err)
	}
	if err := manager.ValidateResources(); err != nil {
		t.Fatalf("ValidateResources() error = %v", err)
	}

	for namespace, dir := range resources {
		for _, locale := range SupportedLanguagesStringSlice {
			path := filepath.Join(dir, locale+".yaml")
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s locale %s: %v", namespace, locale, err)
			}
			assertNoMojibake(t, path, string(raw))
			values := map[string]string{}
			if err := yaml.Unmarshal(raw, &values); err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for key, value := range values {
				if strings.TrimSpace(value) == "" {
					t.Fatalf("%s has empty i18n value for %q", path, key)
				}
			}
		}
	}

	for _, path := range []string{
		filepath.Join(root, "web", "app", "app", "i18n", "locales", "zh-CN.json"),
		filepath.Join(root, "web", "app", "app", "i18n", "locales", "en.json"),
	} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read WebUI locale %s: %v", path, err)
		}
		assertNoMojibake(t, path, string(raw))
		var decoded any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			t.Fatalf("parse WebUI locale %s: %v", path, err)
		}
	}
}

func writeLocale(t *testing.T, root, namespace, locale, content string) {
	t.Helper()
	dir := filepath.Join(root, namespace)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir locale dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, locale+".yaml"), []byte(content), 0o600); err != nil {
		t.Fatalf("write locale: %v", err)
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if info, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil && !info.IsDir() {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repository root not found")
		}
		dir = parent
	}
}

func assertNoMojibake(t *testing.T, path string, text string) {
	t.Helper()
	for _, marker := range []string{"�", "Ã", "Â", "â", "??"} {
		if strings.Contains(text, marker) {
			t.Fatalf("%s contains mojibake marker %q", path, marker)
		}
	}
}
