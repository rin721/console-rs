package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestApplyPrivacyUpdatesPersistsSecretsAndRejectsEnvManagedFields(t *testing.T) {
	configPath := copyWritablePrivacyConfig(t)
	if err := ApplyPrivacyUpdates(configPath, map[string]string{
		"auth.signing_key":          "updated-signing-secret-at-least-32-bytes",
		"auth.refresh_token_pepper": "",
		"unsupported.path":          "ignored",
	}); err != nil {
		t.Fatalf("ApplyPrivacyUpdates() error = %v", err)
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read updated config: %v", err)
	}
	text := string(content)
	if !strings.Contains(text, `signing_key: "updated-signing-secret-at-least-32-bytes"`) {
		t.Fatalf("updated config missing persisted signing key:\n%s", text)
	}
	if strings.Contains(text, "unsupported.path") {
		t.Fatalf("unsupported path should not be persisted:\n%s", text)
	}

	t.Setenv("AUTH_SIGNING_KEY", "managed-by-env-signing-secret-at-least-32-bytes")
	err = ApplyPrivacyUpdates(configPath, map[string]string{
		"auth.signing_key": "another-secret-at-least-32-bytes",
	})
	if err == nil || !strings.Contains(err.Error(), "environment variable AUTH_SIGNING_KEY") {
		t.Fatalf("expected env-managed field error, got %v", err)
	}
}

func copyWritablePrivacyConfig(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "configs", "config.example.yaml"))
	if err != nil {
		t.Fatalf("read config example: %v", err)
	}
	text := strings.Replace(
		string(raw),
		"signing_key: ${AUTH_SIGNING_KEY:dev-signing-key-change-me-32-bytes}",
		"signing_key: writable-signing-secret-at-least-32-bytes",
		1,
	)
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatalf("write writable temp config: %v", err)
	}
	return path
}
