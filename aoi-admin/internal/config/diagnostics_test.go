package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadDiagnosticsCollectsProductionBlockingItems(t *testing.T) {
	unsetEnvNamesForConfigPaths(t,
		"database.driver",
		"database.postgres.host",
		"database.postgres.username",
		"database.postgres.database",
		"auth.signing_key",
		"auth.refresh_token_pepper",
		"auth.mfa_secret_key",
		"auth.notification_driver",
		"auth.smtp.host",
		"auth.smtp.from",
	)
	configPath := copyProductionExampleConfig(t)

	_, diagnostics, err := LoadDiagnostics(configPath)
	if err != nil {
		t.Fatalf("LoadDiagnostics() error = %v", err)
	}

	for _, want := range []string{
		"database.postgres.host",
		"database.postgres.username",
		"database.postgres.database",
		"auth.signing_key",
		"auth.refresh_token_pepper",
		"auth.mfa_secret_key",
		"auth.smtp.host",
		"auth.smtp.from",
	} {
		if !diagnosticPathsContain(diagnostics, want) {
			t.Fatalf("diagnostics missing %q:\n%#v", want, diagnostics)
		}
	}
	if !diagnosticEnvNamesContain(diagnostics, "auth.signing_key", "RIN_APP_AUTH_SIGNING_KEY") {
		t.Fatalf("auth.signing_key diagnostic should include env names: %#v", diagnostics)
	}
}

func copyProductionExampleConfig(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	raw, err := os.ReadFile(filepath.Join(root, "deploy", "config.production.example.yaml"))
	if err != nil {
		t.Fatalf("read production config example: %v", err)
	}
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write temp config: %v", err)
	}
	return path
}

func unsetEnvNamesForConfigPaths(t *testing.T, paths ...string) {
	t.Helper()
	for _, path := range paths {
		for _, key := range EnvNamesForPath(path) {
			key := key
			oldValue, existed := os.LookupEnv(key)
			if err := os.Unsetenv(key); err != nil {
				t.Fatalf("unset %s: %v", key, err)
			}
			t.Cleanup(func() {
				if existed {
					if err := os.Setenv(key, oldValue); err != nil {
						t.Errorf("restore %s: %v", key, err)
					}
					return
				}
				if err := os.Unsetenv(key); err != nil {
					t.Errorf("restore unset %s: %v", key, err)
				}
			})
		}
	}
}

func diagnosticPathsContain(diagnostics []ConfigDiagnostic, path string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Path == path {
			return true
		}
	}
	return false
}

func diagnosticEnvNamesContain(diagnostics []ConfigDiagnostic, path string, envName string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Path != path {
			continue
		}
		for _, candidate := range diagnostic.EnvNames {
			if strings.EqualFold(candidate, envName) {
				return true
			}
		}
	}
	return false
}
